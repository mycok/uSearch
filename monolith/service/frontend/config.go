package frontend

import (
	"fmt"
	"html/template"
	"io"

	"github.com/hashicorp/go-multierror"
	"github.com/sirupsen/logrus"

	"github.com/mycok/uSearch/linkgraph/graph"
	"github.com/mycok/uSearch/textindexer/index"
)

const (
	defaultNumOfResultsPerPage = 10
	defaultMaxSummaryLength    = 256
)

// GraphAPI defines a minimum set of API methods for adding and updating links in
// the link graph store.
type GraphAPI interface {
	// UpsertLink creates a new or updates an existing link.
	UpsertLink(link *graph.Link) error
}

// IndexAPI defines a minimum set of API methods for searching indexed documents.
type IndexAPI interface {
	// Search performs a look up based on query and returns a result
	// iterator if successful or an error otherwise.
	Search(q index.Query) (index.Iterator, error)
}

// Config defines configurations for the front-end service.
type Config struct {
	// API for inserting and updating links in the data store.
	GraphAPI GraphAPI

	// API for searching the index store.
	IndexAPI IndexAPI

	// Port to listen for incoming requests.
	ListenAddr string

	// Number of results per page. If not specified, a default value of 10 results
	// per page will be used instead.
	NumOfResultsPerPage int

	// The maximum length (in characters) of the highlighted content summary for
	// matching documents. If not specified, a default value of 256 will be used
	// instead.
	MaxSummaryLength int

	// The logger to use. If not defined an output-discarding logger will
	// be used instead.
	Logger *logrus.Entry

	// Cache for all application templates.
	TemplateCache map[string]*template.Template
}

func (config *Config) validate() error {
	var err error

	if config.GraphAPI == nil {
		err = multierror.Append(err, fmt.Errorf("graph API not provided"))
	}

	if config.IndexAPI == nil {
		err = multierror.Append(err, fmt.Errorf("index API not provided"))
	}

	if config.ListenAddr == "" {
		err = multierror.Append(err, fmt.Errorf("listen address not provided"))
	}

	if config.NumOfResultsPerPage <= 0 {
		config.NumOfResultsPerPage = defaultNumOfResultsPerPage
	}

	if config.MaxSummaryLength <= 0 {
		config.MaxSummaryLength = defaultMaxSummaryLength
	}

	if config.Logger == nil {
		config.Logger = logrus.NewEntry(&logrus.Logger{Out: io.Discard})
	}

	return err
}
