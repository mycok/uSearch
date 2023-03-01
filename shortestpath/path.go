/*
	shortestpath package provides a custom implementation of
	Dijkstra's algorithm for finding the shortest path in an un-directed graph.
*/

package shortestpath

import (
	"context"
	"fmt"
	"math"

	"github.com/mycok/uSearch/bsp"
	"github.com/mycok/uSearch/bsp/queue"
)

// Static and compile-time check to ensure pathCostMessage implements
// the Message interface.
var _ queue.Message = (*pathCostMessage)(nil)

type pathCostMessage struct {
	// ID of the vertex from which this cost message originates.
	fromID string
	// Cost of the path from this vertex to the source vertex via FromID.
	cost int
}

// Type returns the type of this message.
func (m pathCostMessage) Type() string { return "cost" }

// pathState represents information about a vertex path and is stored in the
// value field on a vertex.
type pathState struct {
	// Distance from the closest vertex / node.
	minDistance int
	// ID of the last / previous closest vertex / node from which the
	// above minDistance was computed.
	prevInPath string
}

// Calculator ia a shortest path calculator from a single vertex to
// all other vertices in a connected graph.
type Calculator struct {
	g *bsp.Graph
	// A vertex from which to start the calculation.
	srcID string
	// Executor to manage step execution. it's is lazily constructed.
	executorFactory bsp.ExecutorFactory
}

// NewCalculator returns a new shortest path calculator.
func NewCalculator(numOfWorkers int) (*Calculator, error) {
	c := &Calculator{
		executorFactory: bsp.NewExecutor,
	}

	g, err := bsp.NewGraph(bsp.GraphConfig{
		ComputeFn:      c.findShortestPath,
		ComputeWorkers: numOfWorkers,
	})
	if err != nil {
		return nil, err
	}

	c.g = g

	return c, nil
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
	c.g.AddVertex(id, nil)
}

// AddEdge adds a directed edge from srcID to dstID with the specified cost.
// An error will be returned if a negative cost value is provided.
func (c *Calculator) AddEdge(srcID string, destID string, cost int) error {
	if cost < 0 {
		return fmt.Errorf("negative edge costs not supported")
	}

	return c.g.AddEdge(srcID, destID, cost)
}

// CalculateShortestPaths executes the shortest path compute function which
// calculates shortest path from the source vertex or node to all other
// vertices / nodes in the graph.
func (c *Calculator) CalculateShortestPaths(
	ctx context.Context, srcID string,
) error {

	c.srcID = srcID
	exec := c.executorFactory(c.g, bsp.ExecutorCallbacks{
		ShouldRunAnotherStep: func(
			ctx context.Context, g *bsp.Graph, activeInStep int,
		) (bool, error) {

			return activeInStep != 0, nil
		},
	})

	return exec.RunToCompletion(ctx)
}

// BuildShortestPathTo builds vertices / nodes that represent the shortest path
// from the source vertex / node to the specified destination.
func (c *Calculator) BuildShortestPathTo(destID string) ([]string, int, error) {
	vertexMap := c.g.Vertices()
	v, exists := vertexMap[destID]
	if !exists {
		return nil, 0, fmt.Errorf("unknown vertex with ID %q", destID)
	}

	var (
		minDistance = v.Value().(*pathState).minDistance
		path        []string
	)

	for ; v.ID() != c.srcID; v = vertexMap[v.Value().(*pathState).prevInPath] {
		path = append(path, v.ID())
	}

	path = append(path, c.srcID)

	// Reverse path slice in place to form path from src->dst
	for i, j := 0, len(path)-1; i < j; i, j = i+1, j-1 {
		path[i], path[j] = path[j], path[i]
	}

	return path, minDistance, nil
}

// findShortestPath serves as the graph compute function. It is mean't to be
// invoked on each vertex / node in the graph.
func (c *Calculator) findShortestPath(
	g *bsp.Graph, v *bsp.Vertex, msgIt queue.Iterator,
) error {

	if g.SuperStep() == 0 {
		v.SetValue(&pathState{minDistance: int(math.MaxInt64)})
	}

	minDistance := int(math.MaxInt64)

	if v.ID() == c.srcID {
		minDistance = 0
	}

	// Process the path-cost messages from neighbors and use the message values
	// to update minDist if the received path cost is better / less than
	// minDistance.
	var viaPath string
	for msgIt.Next() {
		msg := msgIt.Message().(*pathCostMessage)
		if msg.cost < minDistance {
			minDistance = msg.cost
			viaPath = msg.fromID
		}
	}

	// Try to find a better path by processing this vertex. if found, announce
	// it to all neighbors so they can update their own costs.
	state := v.Value().(*pathState)
	// This may not run if this is the first vertex / node and it's also the
	// first step / cycle of computation.
	if minDistance < state.minDistance {
		state.minDistance = minDistance
		state.prevInPath = viaPath

		for _, e := range v.Edges() {
			costMsg := &pathCostMessage{
				fromID: v.ID(),
				cost:   minDistance + e.Value().(int),
			}

			if err := g.SendMessage(e.DestID(), costMsg); err != nil {
				return err
			}
		}
	}

	// Vertex / node is processed and done unless it receive a better path
	// announcement.
	v.Freeze()

	return nil
}
