package shortpath

import (
	"context"
	"fmt"
	"math"

	"github.com/mycok/uSearch/internal/bspgraph"
	"github.com/mycok/uSearch/internal/bspgraph/message"
)

// Calculator implements a shortest path calculator from a single vertex to
// all other vertices in a connected graph.
type Calculator struct {
	g               *bspgraph.Graph // g value must be a pointer of Graph type.
	srcID           string
	executorFactory bspgraph.ExecutorFactory // executorFactory can be a value, a function / method or an expression that returns type *Executor.
}

// NewCalculator returns a new shortest path calculator instance.
func NewCalculator(numOfWorkers int) (*Calculator, error) {
	c := &Calculator{
		executorFactory: bspgraph.NewExecutor,
	}

	var err error
	if c.g, err = bspgraph.NewGraph(bspgraph.GraphConfig{
		ComputeFunc:    c.findShortestPath,
		ComputeWorkers: numOfWorkers,
	}); err != nil {
		return nil, err
	}

	return c, nil
}

// Close cleans up any allocated graph resources.
func (c *Calculator) Close() error {
	return c.g.Close()
}

// SetExecutorFactory configures the calculator to use the a custom executor
// factory when CalculateShortestPaths is invoked.
func (c *Calculator) SetExecutorFactory(factory bspgraph.ExecutorFactory) {
	c.executorFactory = factory
}

// AddVertex inserts a new vertex with the specified ID into the graph.
func (c *Calculator) AddVertex(id string) {
	c.g.AddVertex(id, nil)
}

// AddEdge creates a directed edge from srcID to dstID with the specified cost.
// An error will be returned if a negative cost value is specified.
func (c *Calculator) AddEdge(srcID, destID string, cost int) error {
	if cost < 0 {
		return fmt.Errorf("negative edge costs not supported")
	}

	return c.g.AddEdge(srcID, destID, cost)
}

// CalculateShortestPaths finds the shortest path costs from srcID to all other
// vertices in the graph.
func (c *Calculator) CalculateShortestPaths(ctx context.Context, srcID string) error {
	c.srcID = srcID
	exec := c.executorFactory(c.g, bspgraph.ExecutorCallbacks{
		ShouldPostStepKeepRunning: func(_ context.Context, _ *bspgraph.Graph, activeInStep int) (bool, error) {
			return activeInStep != 0, nil
		},
	})

	return exec.RunToCompletion(ctx)
}

// ShortestPathTo calculates and returns the shortest path from the source vertex to the
// specified destination together with its cost.
func (c *Calculator) ShortestPathTo(destID string) ([]string, int, error) {
	vertMap := c.g.Vertices()
	v, exists := vertMap[destID]
	if !exists {
		return nil, 0, fmt.Errorf("unknown vertex with ID %q", destID)
	}

	var (
		minDist = v.Value().(*pathState).minDistance
		path    []string
	)

	for ; v.ID() != c.srcID; v = vertMap[v.Value().(*pathState).prevInPath] {
		path = append(path, v.ID())
	}

	path = append(path, c.srcID)

	// Reverse in place to get path from src->dst
	for i, j := 0, len(path)-1; i < j; i, j = i+1, j-1 {
		path[i], path[j] = path[j], path[i]
	}

	return path, minDist, nil
}

func (c *Calculator) findShortestPath(g *bspgraph.Graph, v *bspgraph.Vertex, msgIt message.Iterator) error {
	if g.Superstep() == 0 {
		v.SetValue(&pathState{
			minDistance: int(math.MaxInt64),
		})
	}

	minDist := int(math.MaxInt64)

	if v.ID() == c.srcID {
		minDist = 0
	}

	// Process cost messages from neighbors and update minDist if
	// we receive a better path announcement.
	var via string
	for msgIt.Next() {
		m := msgIt.Message().(*PathCostMessage)
		if m.Cost < minDist {
			minDist = m.Cost
			via = m.FromID
		}
	}

	// If a better path is found by processing this vertex, announce it
	// to all neighbors so they can update their own scores.
	state := v.Value().(*pathState)
	if minDist < state.minDistance {
		state.minDistance = minDist
		state.prevInPath = via

		for _, e := range v.Edges() {
			costMsg := &PathCostMessage{
				FromID: v.ID(),
				Cost:   minDist + e.Value().(int),
			}

			if err := g.SendMessage(e.DestID(), costMsg); err != nil {
				return err
			}
		}
	}

	// We are done unless we receive a better path announcement.
	v.Freeze()

	return nil
}

// PathCostMessage is used to advertise the cost of a path through a vertex.
type PathCostMessage struct {
	// The ID of the vertex from which this cost message originates.
	FromID string
	// The cost of the path from this vertex to the source vertex via FromID.
	Cost int
}

// Type returns the type of this message.
func (pc PathCostMessage) Type() string { return "cost" }

// pathState represents info about a vertex path and is stored as the vertex value.
type pathState struct {
	minDistance int
	prevInPath  string
}
