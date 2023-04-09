package pagerank

import (
	"fmt"
	"io"
	"time"

	"github.com/google/uuid"
	"github.com/hashicorp/go-multierror"
	"github.com/juju/clock"
	"github.com/sirupsen/logrus"

	"github.com/mycok/uSearch/linkgraph/graph"
	"github.com/mycok/uSearch/monolith/partition"
)

// GraphAPI defines a minimum set of API methods for querying the link graph store.
type GraphAPI interface {
	// Links returns an iterator for a set of links whose id's belong
	// to the [fromID, toID] range and were retrieved before the [retrievedBefore]
	// time.
	Links(fromID, toID uuid.UUID, retrievedBefore time.Time) (graph.LinkIterator, error)

	// Edges returns an iterator for a set of edges whose source vertex id's
	// belong to the [fromID, toID] range and were updated before the
	// [updatedBefore] time.
	Edges(fromID, toID uuid.UUID, updatedBefore time.Time) (graph.EdgeIterator, error)
}

// IndexAPI defines a minimum set of API methods for updating indexed documents.
type IndexAPI interface {
	// UpdateScore updates the PageRank score for a document with the
	// specified link ID. If no such document exists, a placeholder
	// document with the provided score will be created.
	UpdateScore(linkID uuid.UUID, score float64) error
}

// Config defines configurations for the page-ranking service.
type Config struct {
	// API for querying with the links and edges data store.
	GraphAPI GraphAPI

	// API for communicating with the index store.
	IndexAPI IndexAPI

	// An API for detecting partition assignments for this service.
	PartitionDetector partition.Detector

	// A clock instance for generating time-related events. If not specified,
	// the default wall-clock will be used instead.
	Clock clock.Clock

	// The number of workers to spin up for computing PageRank scores. If
	// not specified, a default value of 1 will be used instead.
	NumOfComputeWorkers int

	// The duration between subsequent page-rank passes.
	UpdateInterval time.Duration

	// The logger to use. If not defined an output-discarding logger will
	// be used instead.
	Logger *logrus.Entry
}

func (config *Config) validate() error {
	var err error

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

	if config.NumOfComputeWorkers <= 0 {
		err = multierror.Append(err, fmt.Errorf("invalid value for compute workers, must be > 0"))
	}

	if config.UpdateInterval == 0 {
		err = multierror.Append(err, fmt.Errorf("invalid value for update interval interval"))
	}

	if config.Logger == nil {
		config.Logger = logrus.NewEntry(&logrus.Logger{Out: io.Discard})
	}

	return err
}
