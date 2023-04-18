package main

import (
	"context"
	"flag"
	"fmt"
	"net/url"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/mycok/uSearch/linkgraph/graph"
	"github.com/mycok/uSearch/linkgraph/store/cdb"
	memgraph "github.com/mycok/uSearch/linkgraph/store/memory"
	"github.com/mycok/uSearch/monolith/partition"
	"github.com/mycok/uSearch/monolith/service"
	"github.com/mycok/uSearch/monolith/service/crawler"
	"github.com/mycok/uSearch/monolith/service/frontend"
	"github.com/mycok/uSearch/monolith/service/pagerank"
	"github.com/mycok/uSearch/textindexer/index"
	"github.com/mycok/uSearch/textindexer/store/es"
	memindex "github.com/mycok/uSearch/textindexer/store/memory"
)

const (
	appName = "uSearch-monolith"
	appSHA  = "compiled-and-deployed-at"
)

func main() {
	host, _ := os.Hostname()
	// Instantiate a root logger that will be passed to all services.
	rootLogger := logrus.New()
	logger := rootLogger.WithFields(logrus.Fields{
		"app":  appName,
		"SHA":  appSHA,
		"host": host,
	})

	svcGroup, err := configureServices(logger)
	if err != nil {
		logger.WithField("err", err).Error("shutting down due to an error")

		return
	}

	ctx, cancelFn := context.WithCancel(context.Background())
	defer cancelFn()

	// Launch a separate process to listen and respond to os signals
	// and trigger a graceful shutdown.
	go func() {
		signalChan := make(chan os.Signal, 1)
		signal.Notify(signalChan, syscall.SIGINT, syscall.SIGHUP)

		select {
		case s := <-signalChan:
			logger.WithField("signal", s.String()).Info("shutting down due to os signal")
			// Cancel context, this signals all services to return since they all
			// share this same context.
			cancelFn()
		case <-ctx.Done():
		}
	}()

	if err := svcGroup.Execute(ctx); err != nil {
		logger.WithField("err", err).Error("shutting down due to an error")

		return
	}

	// Shutdown due to context cancellation.
	logger.Info("shutdown complete")
}

func configureServices(logger *logrus.Entry) (service.Group, error) {
	var (
		crawlerConfig  crawler.Config
		frontendConfig frontend.Config
		pageRankConfig pagerank.Config
	)

	flag.IntVar(
		&crawlerConfig.NumOfFetchWorkers, "crawler-num-workers",
		runtime.NumCPU(),
		"Number of workers for crawling web pages.[defaults to number of CPU's]",
	)
	flag.DurationVar(
		&crawlerConfig.CrawlUpdateInterval, "crawler-update-interval",
		5*time.Minute, "Time between subsequent crawler runs",
	)
	flag.DurationVar(
		&crawlerConfig.ReIndexThreshold, "crawler-re-index-threshold",
		7*24*time.Hour,
		"Minimum amount of time before re-indexing an already crawled[indexed] link",
	)

	flag.StringVar(
		&frontendConfig.ListenAddr, "frontend-listen-addr",
		":8080", "Address to listen on for incoming frontend requests",
	)
	flag.IntVar(
		&frontendConfig.NumOfResultsPerPage, "frontend-search-results-per-page",
		10, "Number of search results displayed per page",
	)
	flag.IntVar(
		&frontendConfig.MaxSummaryLength, "frontend-max-summary-length",
		256, "The maximum length of the summary for each matched document in characters",
	)

	flag.IntVar(
		&pageRankConfig.NumOfComputeWorkers, "pagerank-num-workers",
		runtime.NumCPU(), "Number of workers for computing page ranks",
	)
	flag.DurationVar(
		&pageRankConfig.UpdateInterval, "pagerank-update-interval",
		time.Hour, "Time between subsequent page rank score updates",
	)

	linkGraphURI := flag.String(
		"link-graph-uri", "in-memory://",
		"URI for connecting to a link-graph data store."+
			" [supported URI's: in-memory://, postgresql://user@host:26257/linkgraph?sslmode=disable]",
	)
	textIndexURI := flag.String(
		"text-index-uri", "in-memory://",
		"URI for connecting to a text-index data store."+
			" [supported URI's: in-memory://, es://node1:9200,...,nodeN:9200]",
	)

	partitionDetectorMode := flag.String(
		"partition-detection-mode", "single",
		"The partition detection mode to use. Supported values are"+
			" 'dns=HEADLESS_SERVICE_NAME' (k8s) and 'single' (local dev mode)",
	)

	flag.Parse()

	// Retrieve a suitable link graph and text indexer implementation and
	// plug it into service configurations.
	linkGraph, err := getLinkGraph(*linkGraphURI, logger)
	if err != nil {
		return nil, err
	}

	textIndex, err := getTextIndex(*textIndexURI, logger)
	if err != nil {
		return nil, err
	}

	partDet, err := getPartitionDetector(*partitionDetectorMode)
	if err != nil {
		return nil, err
	}

	var svc service.Service
	var svcGrp service.Group

	crawlerConfig.GraphAPI = linkGraph
	crawlerConfig.IndexAPI = textIndex
	crawlerConfig.PartitionDetector = partDet
	crawlerConfig.Logger = logger.WithField("service", "crawler")
	if svc, err = crawler.New(crawlerConfig); err == nil {
		svcGrp = append(svcGrp, svc)
	} else {
		return nil, err
	}

	frontendConfig.GraphAPI = linkGraph
	frontendConfig.IndexAPI = textIndex
	frontendConfig.Logger = logger.WithField("service", "frontend")
	if svc, err = frontend.New(frontendConfig); err == nil {
		svcGrp = append(svcGrp, svc)
	} else {
		return nil, err
	}

	pageRankConfig.GraphAPI = linkGraph
	pageRankConfig.IndexAPI = textIndex
	pageRankConfig.PartitionDetector = partDet
	pageRankConfig.Logger = logger.WithField("service", "page-rank-calculator")
	if svc, err = pagerank.New(pageRankConfig); err == nil {
		svcGrp = append(svcGrp, svc)
	} else {
		return nil, err
	}

	return svcGrp, nil
}

// GraphAPI defines a minimum set of API methods for the link graph data store.
type GraphAPI interface {
	// UpsertLink creates a new or updates an existing link.
	UpsertLink(link *graph.Link) error

	// UpsertEdge creates a new or updates an existing edge.
	UpsertEdge(edge *graph.Edge) error

	// Links returns an iterator for a set of links whose id's belong
	// to the [fromID, toID] range and were retrieved before the [retrievedBefore]
	// time.
	Links(fromID, toID uuid.UUID, retrievedBefore time.Time) (graph.LinkIterator, error)

	// Edges returns an iterator for a set of edges whose source vertex id's
	// belong to the [fromID, toID] range and were updated before the
	// [updatedBefore] time.
	Edges(fromID, toID uuid.UUID, updatedBefore time.Time) (graph.EdgeIterator, error)

	// RemoveStaleEdges removes any edge that originates from a specific link ID
	// and was updated before the specified [updatedBefore] time.
	RemoveStaleEdges(fromID uuid.UUID, updatedBefore time.Time) error
}

func getLinkGraph(linkGraphURI string, logger *logrus.Entry) (GraphAPI, error) {
	if linkGraphURI == "" {
		return nil, fmt.Errorf("link graph URI must be specified with --link-graph-uri")
	}

	url, err := url.Parse(linkGraphURI)
	if err != nil {
		return nil, fmt.Errorf("failed to parse link graph URI: %w", err)
	}

	switch url.Scheme {
	case "in-memory":
		logger.Info("using in-memory link graph store")

		return memgraph.NewInMemoryGraph(), nil
	case "postgresql":
		logger.Info("using CDB link graph store")

		return cdb.NewCockroachDBGraph(linkGraphURI)
	default:
		return nil, fmt.Errorf("unsupported link graph URI scheme: %q", url.Scheme)
	}
}

// IndexAPI defines a minimum set of API methods for searching indexed documents.
type IndexAPI interface {
	// Index adds a new document or updates an existing index entry
	// in case of an existing document.
	Index(doc *index.Document) error

	// Search performs a look up based on query and returns a result
	// iterator if successful or an error otherwise.
	Search(q index.Query) (index.Iterator, error)

	// UpdateScore updates the PageRank score for a document with the
	// specified link ID. If no such document exists, a placeholder
	// document with the provided score will be created.
	UpdateScore(linkID uuid.UUID, score float64) error
}

func getTextIndex(textIndexURI string, logger *logrus.Entry) (IndexAPI, error) {
	if textIndexURI == "" {
		return nil, fmt.Errorf("text index URI must be specified with --text-index-uri")
	}

	url, err := url.Parse(textIndexURI)
	if err != nil {
		return nil, fmt.Errorf("failed to parse text index URI: %w", err)
	}

	switch url.Scheme {
	case "in-memory":
		logger.Info("using in-memory index store")

		return memindex.NewInMemoryIndex()
	case "es":
		nodes := strings.Split(url.Host, ",")
		for i := 0; i < len(nodes); i++ {
			nodes[i] = "http://" + nodes[i]
		}
		logger.Info("using ES index store")

		return es.NewEsIndexer(nodes, false)
	default:
		return nil, fmt.Errorf("unsupported text index URI scheme: %q", url.Scheme)
	}
}

func getPartitionDetector(mode string) (partition.Detector, error) {
	switch {
	case mode == "single":
		return partition.DummyDetector{
			Partition:       0,
			NumOfPartitions: 1,
		}, nil
	case strings.HasPrefix(mode, "dns="):
		tokens := strings.Split(mode, "=")
		return partition.DetectFromSRVRecords(tokens[1]), nil
	default:
		return nil, fmt.Errorf("unsupported partition detector mode: %q", mode)
	}
}
