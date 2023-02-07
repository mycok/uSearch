package bsp

import (
	"errors"

	"github.com/hashicorp/go-multierror"

	"github.com/mycok/uSearch/bsp/queue"
)

// GraphConfig encapsulates the configuration options for creating graphs.
type GraphConfig struct {
	// QueueFactory is used by the graph to create message queue instances
	// for each vertex that is added to the graph. If not specified, the
	// default in-memory queue will be used instead.
	QueueFactory queue.Factory

	// ComputeFn is the compute function that will be invoked for each graph
	// vertex when executing a superStep. A valid ComputeFunc instance is
	// required for the config to be valid.
	ComputeFn ComputeFunc

	// ComputeWorkers specifies the number of workers to use for invoking
	// the registered ComputeFunc when executing each superStep. If not
	// specified, a single worker will be used.
	ComputeWorkers int
}

// Validate checks whether a graph configuration is valid and sets the default
// values if required.
func (c *GraphConfig) Validate() error {
	var err error

	if c.QueueFactory == nil {
		c.QueueFactory = queue.NewInMemoryQueue
	}

	if c.ComputeFn == nil {
		err = multierror.Append(err, errors.New("compute function not provided"))
	}

	if c.ComputeWorkers <= 0 {
		c.ComputeWorkers = 1
	}

	return err
}
