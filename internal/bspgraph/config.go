package bspgraph

import (
	"errors"

	"github.com/mycok/uSearch/internal/bspgraph/message"

	"github.com/hashicorp/go-multierror"
)

// GraphConfig encapsulates the configuration options for creating graphs.
type GraphConfig struct {
	// QueueFactory is used by the graph to create message queue instances
	// for each vertex that is added to the graph. If not specified, the
	// default in-memory queue will be used instead.
	QueueFactory message.QueueFactory

	// ComputeFn is the compute function that will be invoked for each graph
	// vertex when executing a superstep. A valid ComputeFunc instance is
	// required for the config to be valid.
	ComputeFunc ComputeFunc

	// ComputeWorkers specifies the number of workers to use for invoking
	// the registered ComputeFunc when executing each superstep. If not
	// specified, a single worker will be used.
	ComputeWorkers int
}

// validate checks whether a graph configuration is valid and sets the default
// values where required.
func (g *GraphConfig) Validate() error {
	var err error

	if g.QueueFactory == nil {
		g.QueueFactory = message.NewInMemoryQueue
	}

	if g.ComputeFunc == nil {
		err = multierror.Append(err, errors.New("compute function not specified"))
	}

	if g.ComputeWorkers <= 0 {
		g.ComputeWorkers = 1
	}

	return err
}
