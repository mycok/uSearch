package pagerank

import (
	"context"
	"fmt"

	"github.com/mycok/uSearch/bsp"
	"github.com/mycok/uSearch/bsp/aggregator"
)

// Calculator executes the iterative version of the PageRank algorithm
// on a graph until the desired level of convergence is reached.
type Calculator struct {
	g               *bsp.Graph
	cfg             Config
	executorFactory bsp.ExecutorFactory
}

// NewCalculator returns a new Calculator instance using the provided config
// options.
func NewCalculator(cfg Config) (*Calculator, error) {
	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf(
			"PageRank calculator config validation failed: %w", err,
		)
	}

	g, err := bsp.NewGraph(bsp.GraphConfig{
		ComputeWorkers: cfg.ComputeWorkers,
		ComputeFn:      makeComputeFunc(cfg.DampingFactor),
	})
	if err != nil {
		return nil, err
	}

	return &Calculator{
		g:               g,
		cfg:             cfg,
		executorFactory: bsp.NewExecutor,
	}, nil
}

// Graph returns the underlying Graph instance.
func (c *Calculator) Graph() *bsp.Graph {
	return c.g
}

// Close frees up any allocated graph resources.
func (c *Calculator) Close() error {
	return c.g.Close()
}

// SetExecutorFactory sets a custom executor factory for the calculator.
func (c *Calculator) SetExecutorFactory(factory bsp.ExecutorFactory) {
	c.executorFactory = factory
}

// AddVertex adds a new vertex / node with the specified ID into the graph.
func (c *Calculator) AddVertex(id string) {
	c.g.AddVertex(id, 0.0)
}

// AddEdge inserts a directed edge from src to dst. If both src and dst refer
// to the same vertex then this is a no-op.
func (c *Calculator) AddEdge(src, dst string) error {
	// Don't allow self-links
	if src == dst {
		return nil
	}
	return c.g.AddEdge(src, dst, nil)
}

// Scores invokes the provided visitor function for each vertex in the graph.
func (c *Calculator) Scores(visitFn func(id string, score float64) error) error {
	for id, v := range c.g.Vertices() {
		if err := visitFn(id, v.Value().(float64)); err != nil {
			return err
		}
	}

	return nil
}

// CalculatePageRanks executes the compute function which uses the page rank
// algorithm to calculate page ranks for all the vertices / pages in the graph.
func (c *Calculator) CalculatePageRanks(ctx context.Context) error {
	c.registerAggregators()

	exec := c.executorFactory(c.g, bsp.ExecutorCallbacks{
		PreStep: func(_ context.Context, g *bsp.Graph) error {
			// Reset sum of abs differences aggregator and residual
			// aggregator for next step.
			g.Aggregator("SAD").Set(0.0)
			g.Aggregator(residualOutputAccName(g.SuperStep())).Set(0.0)

			return nil
		},

		ShouldRunAnotherStep: func(
			_ context.Context, g *bsp.Graph, _ int,
		) (bool, error) {

			// Since super steps 0 and 1 are part of the algorithm
			// initialization, the predicate should only be evaluated for
			// super steps > 1
			//
			// sum of absolute differences [SAD]
			sad := c.g.Aggregator("SAD").Get().(float64)

			return !(g.SuperStep() > 1 && sad < c.cfg.MinSADForConvergence), nil
		},
	})

	return exec.RunToCompletion(ctx)
}

// registerAggregators creates and registers the aggregator instances that we
// need to run the PageRank calculation algorithm.
func (c *Calculator) registerAggregators() {
	c.g.RegisterAggregator("page_count", new(aggregator.IntAccumulator))
	c.g.RegisterAggregator("residual_0", new(aggregator.Float64Accumulator))
	c.g.RegisterAggregator("residual_1", new(aggregator.Float64Accumulator))
	c.g.RegisterAggregator("SAD", new(aggregator.Float64Accumulator))
}
