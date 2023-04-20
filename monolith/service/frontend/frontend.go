package frontend

import (
	"bytes"
	"context"
	"embed"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"net"
	"net/http"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/sirupsen/logrus"

	"github.com/mycok/uSearch/linkgraph/graph"
	"github.com/mycok/uSearch/textindexer/index"
)

//go:embed ui/*
var templateFS embed.FS

const (
	indexEndpoint      = "/"
	searchEndpoint     = "/search"
	submitLinkEndpoint = "/submit/site"
)

// Service represents a front-end service for the uSearch application. it
// satisfies the service.Service interface.
type Service struct {
	config Config
	// Any router type that satisfies the http.Handler interface.
	router *chi.Mux
	// A template executor hook that tests can override.
	templExecutor func(
		templ *template.Template, w io.Writer, data map[string]interface{},
	) error
}

// New creates and returns a fully configured page-rank service instance.
func New(config Config) (*Service, error) {
	if err := config.validate(); err != nil {
		return nil, fmt.Errorf("frontend service: config validation failed: %w", err)
	}

	svc := &Service{
		config: config,
		router: chi.NewRouter(),
		templExecutor: executeTemplate,
	}

	svc.router.Get(indexEndpoint, svc.renderIndexPage)
	svc.router.Get(searchEndpoint, svc.renderSearchResults)
	svc.router.Get(submitLinkEndpoint, svc.submitLink)
	svc.router.Post(submitLinkEndpoint, svc.submitLink)

	fileServer := http.FileServer(http.FS(templateFS))
	svc.router.Handle("/ui/static/*", fileServer)

	svc.router.NotFound(http.HandlerFunc(svc.render404Page))

	return svc, nil
}

// Name returns the name of the service.
func (svc *Service) Name() string { return "frontend" }

// Run executes the service and blocks until the context gets cancelled
// or an error occurs.
func (svc *Service) Run(ctx context.Context) error {
	l, err := net.Listen("tcp", svc.config.ListenAddr)
	if err != nil {
		return err
	}
	defer func() { _ = l.Close() }()

	// Compile and cache frontend service templates.
	if err := svc.newTemplateCache(); err != nil {
		return err
	}

	srv := &http.Server{
		Addr:    svc.config.ListenAddr,
		Handler: svc.router,
	}

	go func() {
		<-ctx.Done()
	
		_ = srv.Close()
	}()

	svc.config.Logger.WithField("addr", svc.config.ListenAddr).Info(
		"started service",
	)

	if err = srv.Serve(l); err == http.ErrServerClosed {
		// Server closed gracefully.
		err = nil
	}

	return err
}

func (svc *Service) newTemplateCache() error {
	cache := make(map[string]*template.Template)

	// Get a slice of all file paths matching a '*.comp.go.tmpl' pattern. 
	components, err := fs.Glob(templateFS, "ui/html/*.comp.go.tmpl")
	if err != nil {
		return err
	}

	var name string

	for _, comp := range components {
		// Extract the file name [ie 'home.page.go.tmpl'] from the full file path
		// and assign it to the name variable.
		name = filepath.Base(comp)
		t, err := template.New(name).ParseFS(templateFS, comp)
		if err != nil {
			return err
		}

		// Parse the newly created template to add any 'layout' templates.
		t, err = t.ParseFS(templateFS, "ui/html/" + "comp.layout.go.tmpl")
		if err != nil {
			return err
		}

		// Parse the newly created template to add any 'partial' templates.
		t, err = t.ParseFS(templateFS, "ui/html/" + "footer.partial.go.tmpl")
		if err != nil {
			return err
		}


		// Add the template to the cache, using the file name (like 'home.page.go.tmpl')
		//  as the key.
		cache[name] = t
	}

	pages, err := fs.Glob(templateFS, "ui/html/*.page.go.tmpl")
	if err != nil {
		return err
	}

	for _, page := range pages {
		name = filepath.Base(page)
		t, err := template.New(name).ParseFS(templateFS, page)
		if err != nil {
			return err
		}

		t, err = t.ParseFS(templateFS, "ui/html/" + "page.layout.go.tmpl")
		if err != nil {
			return err
		}

		t, err = t.ParseFS(templateFS, "ui/html/" + "footer.partial.go.tmpl")
		if err != nil {
			return err
		}

		cache[name] = t

	}

	svc.config.TemplateCache = cache

	return nil
}

func (svc *Service) render404Page(w http.ResponseWriter, _ *http.Request) {
	err := svc.templExecutor(svc.config.TemplateCache["error.comp.go.tmpl"], w, map[string]interface{}{
		"indexEndpoint":  indexEndpoint,
		"searchEndpoint": searchEndpoint,
		"searchTerms":    "",
		"messageTitle":   "Page not found",
		"messageContent": "Page not found",
	})

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func (svc *Service) renderIndexPage(w http.ResponseWriter, _ *http.Request) {
	err := svc.templExecutor(svc.config.TemplateCache["index.page.go.tmpl"], w, map[string]interface{}{
		"indexEndpoint":  indexEndpoint,
		"searchEndpoint":     searchEndpoint,
		"submitLinkEndpoint": submitLinkEndpoint,
	})

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func (svc *Service) submitLink(w http.ResponseWriter, r *http.Request) {
	var msg string

	defer func() {
		err := svc.templExecutor(svc.config.TemplateCache["submit.page.go.tmpl"], w, map[string]interface{}{
			"indexEndpoint":          indexEndpoint,
			"submitLinkEndpoint": submitLinkEndpoint,
			"messageContent":     msg,
		})

		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
		}
	}()

	if r.Method == "POST" {
		// Parse and read the request url query.
		if err := r.ParseForm(); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			msg = "Invalid or malformed link"

			return
		}

		link, err := url.Parse(r.Form.Get("link"))
		if err != nil || (link.Scheme != "http" && link.Scheme != "https") {
			w.WriteHeader(http.StatusBadRequest)
			msg = "Invalid or malformed link"

			return
		}

		link.Fragment = ""
		if err = svc.config.GraphAPI.UpsertLink(
			&graph.Link{URL: link.String()},
		); err != nil {
			svc.config.Logger.WithField("err", err).Errorf(
				"could not upsert link into link graph",
			)

			w.WriteHeader(http.StatusInternalServerError)
			msg = "An error occurred while adding web site to the index; please try again later."

			return
		}

		msg = "Website link successfully submitted"
	}
}

func (svc *Service) renderSearchResults(w http.ResponseWriter, r *http.Request) {
	searchTerms := r.URL.Query().Get("q")
	offset, _ := strconv.ParseUint(r.URL.Query().Get("offset"), 10, 64)

	matchedDocs, pagination, err := svc.runQuery(searchTerms, offset)
	if err != nil {
		svc.config.Logger.WithField("err", err).Error("search query execution failed")
		svc.renderSearchErrorPage(w, searchTerms)

		return
	}

	if err := svc.templExecutor(svc.config.TemplateCache["results.comp.go.tmpl"], w, map[string]interface{}{
		"indexEndpoint":  indexEndpoint,
		"searchEndpoint": searchEndpoint,
		"searchTerms":    searchTerms,
		"pagination":     pagination,
		"results":        matchedDocs,
	}); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func (svc *Service) runQuery(searchTerms string, offset uint64) (
	[]matchedDoc, *paginationDetails, error,
) {

	query := index.Query{
		Type:       index.QueryTypeMatch,
		Expression: searchTerms,
		Offset:     offset,
	}

	if strings.HasPrefix(searchTerms, `"`) && strings.HasSuffix(searchTerms, `"`) {
		query.Type = index.QueryTypePhrase
		searchTerms = strings.Trim(searchTerms, `"`)
	}

	docsIt, err := svc.config.IndexAPI.Search(query)
	if err != nil {
		return nil, nil, err
	}
	defer func() { _ = docsIt.Close() }()

	// Wrap every doc in a matchedDoc object and generate a short summary
	// which also highlights the matched search terms.
	matchedDocs := make([]matchedDoc, 0, svc.config.NumOfResultsPerPage)
	summarizer := newMatchSummarizer(searchTerms, svc.config.MaxSummaryLength)
	highlighter := newMatchHighlighter(searchTerms)

	for docCount := 0; docsIt.Next() && docCount < svc.config.NumOfResultsPerPage; docCount++ {
		doc := docsIt.Document()
		matchedDocs = append(matchedDocs, matchedDoc{
			doc: doc,
			summary: highlighter.Highlight(
				template.HTMLEscapeString(
					summarizer.Summary(doc.Content),
				),
			),
		})
	}

	if err = docsIt.Error(); err != nil {
		return nil, nil, err
	}

	// Setup pagination and generate prev/next links.
	pagination := &paginationDetails{
		From:  int(offset + 1),
		To:    int(offset) + len(matchedDocs),
		Total: int(docsIt.TotalCount()),
	}

	if offset > 0 {
		pagination.PrevLink = fmt.Sprintf("%s?q=%s", searchEndpoint, searchTerms)
		prevOffset := int(offset) - len(matchedDocs)
		if prevOffset > 0 {
			pagination.PrevLink += fmt.Sprintf("&offset=%d", offset)
		}
	}

	nextPageOffset := int(offset) + len(matchedDocs)
	if nextPageOffset < pagination.Total {
		pagination.NextLink = fmt.Sprintf(
			"%s?q=%s&offset=%d",
			searchEndpoint, searchTerms, nextPageOffset,
		)
	}

	return matchedDocs, pagination, nil
}

func (svc *Service) renderSearchErrorPage(w http.ResponseWriter, searchTerms string) {
	w.WriteHeader(http.StatusInternalServerError)

	_ = svc.templExecutor(svc.config.TemplateCache["error.comp.go.tmpl"], w, map[string]interface{}{
		"indexEndpoint":  indexEndpoint,
		"searchEndpoint": searchEndpoint,
		"searchTerms":    searchTerms,
		"messageTitle":   "Error",
		"messageContent": "An error occurred, please try again later.",
	})
}

func executeTemplate(
	templ *template.Template, w io.Writer, data map[string]interface{},
) error {

	// initialize a new buffer and write the template to the buffer
	// instead of straight to the response writer. incase of an error,
	// it's immediately returned.
	buff := new(bytes.Buffer)

	if err := templ.Execute(buff, data); err != nil {
		logrus.Errorf("failed to execute template: %s: %s", templ.Name(), err.Error())

		return err
	}

	buff.WriteTo(w)

	return nil
}

// paginationDetails encapsulates the details for rendering a paginator component.
type paginationDetails struct {
	From     int
	To       int
	Total    int
	PrevLink string
	NextLink string
}

// matchedDoc wraps an index.Document and provides convenience methods for
// rendering its contents in a search results view.
type matchedDoc struct {
	doc     *index.Document
	summary string
}

// HighlightedSummary returns a highlighted document summary as an HTML
// template.
func (d *matchedDoc) HighlightedSummary() template.HTML {
	return template.HTML(d.summary)
}

// Returns the document URL.
func (d *matchedDoc) URL() string {
	return d.doc.URL
}

// Returns the document title.
func (d *matchedDoc) Title() string {
	if d.doc.Title != "" {
		return d.doc.Title
	}

	return d.doc.URL
}
