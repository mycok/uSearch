package crawler

import (
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/hashicorp/go-multierror"
	"github.com/juju/clock"
	"github.com/sirupsen/logrus"

	"github.com/mycok/uSearch/crawler"
	"github.com/mycok/uSearch/crawler/privnet"
	"github.com/mycok/uSearch/linkgraph/graph"
	"github.com/mycok/uSearch/monolith/partition"
	"github.com/mycok/uSearch/textindexer/index"
)

// GraphAPI defines a minimum set of API methods for accessing the link graph store.
type GraphAPI interface {
	// UpsertLink creates a new or updates an existing link.
	UpsertLink(link *graph.Link) error

	// UpsertEdge creates a new or updates an existing edge.
	UpsertEdge(edge *graph.Edge) error

	// RemoveStaleEdges removes any edge that originates from a specific link ID
	// and was updated before the specified [updatedBefore] time.
	RemoveStaleEdges(fromID uuid.UUID, updatedBefore time.Time) error

	// Links returns an iterator for a set of links whose id's belong
	// to the [fromID, toID] range and were retrieved before the [retrievedBefore]
	// time.
	Links(fromID, toID uuid.UUID, retrievedBefore time.Time) (graph.LinkIterator, error)
}

// IndexAPI defines a minimum set of API methods for indexing crawled documents.
type IndexAPI interface {
	// Index adds a new document or updates an existing index entry
	// in case of an existing document.
	Index(doc *index.Document) error
}

// Config defines configurations for the web-crawler service.
type Config struct {
	// API for interacting with the links and edges data store. 
	GraphAPI               GraphAPI

	// API for communicating with the index store.
	IndexAPI               IndexAPI

	// An API for detecting private network addresses. If not specified,
	// a default implementation that handles the private network ranges
	// defined in RFC1918 will be used instead.
	PrivateNetworkDetector crawler.PrivateNetworkDetector

	// An API for performing HTTP requests. If not specified,
	// http.DefaultClient will be used instead.
	URLGetter              crawler.URLGetter

	// An API for detecting partition assignments for this service.
	PartitionDetector      partition.Detector

	// A clock instance for generating time-related events. If not specified,
	// the default wall-clock will be used instead.
	Clock                  clock.Clock

	// The number of concurrent workers used for retrieving links.
	NumOfFetchWorkers      int

	// The duration between subsequent crawler passes.
	CrawlUpdateInterval         time.Duration

	// The minimum amount of time before re-indexing an already-crawled link.
	ReIndexThreshold       time.Duration

	// The logger to use. If not defined an output-discarding logger will
	// be used instead.
	Logger                 *logrus.Entry
}

func (config *Config) validate() error {
	var err error

	if config.PrivateNetworkDetector == nil {
		config.PrivateNetworkDetector, err = privnet.NewDetector()
	}

	if config.URLGetter == nil {
		config.URLGetter = http.DefaultClient
	}

	if config.GraphAPI == nil {
		err = multierror.Append(err, fmt.Errorf("graph API not provided"))
	}

	if config.IndexAPI == nil {
		err = multierror.Append(err, fmt.Errorf("index API not provided"))
	}

	if config.PartitionDetector == nil {
		err = multierror.Append(err, fmt.Errorf("partition detector not provided"))
	}

	if config.Clock == nil {
		config.Clock = clock.WallClock
	}

	if config.NumOfFetchWorkers <= 0 {
		err = multierror.Append(err, fmt.Errorf("invalid value for fetch workers, must be > 0"))
	}

	if config.CrawlUpdateInterval == 0 {
		err = multierror.Append(err, fmt.Errorf("invalid value for crawl update interval interval"))
	}

	if config.ReIndexThreshold == 0 {
		err = multierror.Append(err, fmt.Errorf("invalid value for re-index threshold"))
	}

	if config.Logger == nil {
		config.Logger = logrus.NewEntry(&logrus.Logger{Out: io.Discard})
	}

	return err
}
