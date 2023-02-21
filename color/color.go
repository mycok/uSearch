package color

import (
	"context"
	"math/rand"

	"github.com/mycok/uSearch/bsp"
	"github.com/mycok/uSearch/bsp/queue"
)

// Static and compile-time check to ensure colorMessage implements
// the Message interface.
var _ queue.Message = (*colorMessage)(nil)

type colorMessage struct {
	id    string
	token int
	color int
}

// Type returns the type of this message.
func (m colorMessage) Type() string { return "color" }

// colorState represents information about the color used by a vertex and
// is stored in the value field on a vertex.
type colorState struct {
	token int
	color int
	// Keeps track of all colors assigned to the vertex's neighbors.
	usedColors map[int]bool
}

func (c *colorState) asMessage(id string) *colorMessage {
	return &colorMessage{
		id:    id,
		token: c.token,
		color: c.color,
	}
}

// Assigner is a vertex color assigning object.
type Assigner struct {
	g               *bsp.Graph
	executorFactory bsp.ExecutorFactory
}

// NewColorAssigner configures and returns a new color Assigner instance.
func NewColorAssigner(numOfWorkers int) (*Assigner, error) {
	g, err := bsp.NewGraph(bsp.GraphConfig{
		ComputeFn:      assignColorsToGraph,
		ComputeWorkers: numOfWorkers,
	})
	if err != nil {
		return nil, err
	}

	return &Assigner{
		g:               g,
		executorFactory: bsp.NewExecutor,
	}, nil
}

// Graph returns the underlying Graph instance.
func (a *Assigner) Graph() *bsp.Graph {
	return a.g
}

// Close frees up any allocated graph resources.
func (a *Assigner) Close() error {
	return a.g.Close()
}

// SetExecutorFactory sets a custom executor factory for the calculator.
func (a *Assigner) SetExecutorFactory(factory bsp.ExecutorFactory) {
	a.executorFactory = factory
}

// AddVertex adds a new vertex / node with the specified ID into the graph.
func (a *Assigner) AddVertex(id string) {
	a.AddPreColoredVertex(id, 0)
}

// AddPreColoredVertex adds a new vertex / node with the specified ID and color
// into the graph.
func (a *Assigner) AddPreColoredVertex(id string, color int) {
	a.g.AddVertex(id, &colorState{color: color})
}

// AddEdge adds a directed edge from srcID to dstID.
func (a *Assigner) AddEdge(srcID, destID string) error {
	if err := a.g.AddEdge(srcID, destID, nil); err != nil {
		return err
	}

	// Create a reverse edge from the newly created edge as the source to the
	// previous source as destination.
	return a.g.AddEdge(destID, srcID, nil)
}

// AssignColors executes the assigner which in turn invokes
// a user-defined visitor function for each vertex in the graph.
func (a *Assigner) AssignColors(
	ctx context.Context, visitor func(vertexID string, color int),
) (int, error) {

	exec := a.executorFactory(a.g, bsp.ExecutorCallbacks{
		ShouldRunAnotherStep: func(
			_ context.Context, _ *bsp.Graph, activeInStep int,
		) (bool, error) {

			// Stop when all vertices have been colored.
			return activeInStep != 0, nil
		},
	})

	if err := exec.RunToCompletion(ctx); err != nil {
		return 0, nil
	}

	var totalAssignedColors int
	for id, v := range a.g.Vertices() {
		vState := v.Value().(*colorState)
		if vState.color > totalAssignedColors {
			totalAssignedColors = vState.color
		}

		visitor(id, vState.color)
	}

	return totalAssignedColors, nil
}

func assignColorsToGraph(
	g *bsp.Graph, v *bsp.Vertex, msgIt queue.Iterator,
) error {

	v.Freeze()
	vState := v.Value().(*colorState)

	if g.SuperStep() == 0 {
		// If this is the last vertex in a batch, assign it the first possible
		// color and return. This vertex will mark the end of the graph
		// coloring process for subsequent steps since no new messages will be
		// broadcast.
		if vState.color == 0 && len(v.Edges()) == 0 {
			vState.color = 1

			return nil
		}

		vState.token = rand.Int()
		vState.usedColors = make(map[int]bool)

		return g.BroadcastToNeighbors(v, vState.asMessage(v.ID()))
	}

	// During the subsequent steps, if a vertex is already colored
	// (last vertex in the batch) for the second step, return.
	if vState.color != 0 {
		return nil
	}

	// Process and update edge color assignments. Also, figure out if we have
	// the highest token number among the un-colored neighbors so we get to
	// pick the next color.
	//
	// Check if the same vertex token is also assigned to a neighbor, and if so
	// (highly unlikely) then compare the vertex IDs to break the tie.
	shouldPickNextColor := true
	myID := v.ID()

	for msgIt.Next() {
		msg := msgIt.Message().(*colorMessage)
		if msg.color != 0 {
			vState.usedColors[msg.color] = true
		} else if vState.token < msg.token ||
			(vState.token == msg.token) && myID < msg.id {

			shouldPickNextColor = false
		}
	}

	// If it's not yet our turn to pick a color, keep broadcasting our token
	// to each one of our neighbors.
	if !shouldPickNextColor {
		return g.BroadcastToNeighbors(v, vState.asMessage(v.ID()))
	}

	// Find the minimum unused color, assign it to us and announce it to
	// neighbors.
	for nextColor := 1; ; nextColor++ {
		if vState.usedColors[nextColor] {
			continue
		}

		vState.color = nextColor

		return g.BroadcastToNeighbors(v, vState.asMessage(v.ID()))
	}
}
