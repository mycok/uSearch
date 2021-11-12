package graphcolor

import (
	"context"
	"math/rand"

	"github.com/mycok/uSearch/internal/bspgraph"
	"github.com/mycok/uSearch/internal/bspgraph/message"
)

type vertexState struct {
	token int
	color int
	usedColors map[int]bool
}

func (s *vertexState) asMessage(id string) *vertexStateMessage {
	return &vertexStateMessage{
		ID: id,
		Token: s.token,
		Color: s.color,
	}
}

type vertexStateMessage struct {
	ID string
	Token int
	Color int
}

// Type returns the type of the message.
func (m *vertexStateMessage) Type() string { return "vertexStateMessage" }

// Assigner type encapsulates the state of a graph color assigning type.
type Assigner struct {
	g *bspgraph.Graph
	executorFactory bspgraph.ExecutorFactory
}

// NewColorAssigner returns a new color Assigner instance.
func NewColorAssigner(numOfWorkers int) (*Assigner, error) {
	g, err := bspgraph.NewGraph(bspgraph.GraphConfig{
		ComputeFunc: assignColorsToGraph,
		ComputeWorkers: numOfWorkers,
	})
	if err != nil {
		return nil, err
	}

	return &Assigner{
		g: g,
		executorFactory: bspgraph.NewExecutor,
	}, nil
}

// Close releases any allocated graph resources.
func (a *Assigner) Close() error {
	return a.g.Close()
}

// Graph returns the underlying bspgraph.Graph instance.
func (a *Assigner) Graph() *bspgraph.Graph {
	return a.g
}

// SetExecutorFactory configures the calculator to use the a custom executor
// factory when AssignColors is invoked.
func (a *Assigner) SetExecutorFactory(factory bspgraph.ExecutorFactory) {
	a.executorFactory = factory
}

// AddVertex inserts a new vertex with the specified ID into the graph.
func (a *Assigner) AddVertex(id string) {
	a.AddPrecoloredVertex(id, 0)
}

// AddPrecoloredVertex inserts a new vertex with a pre-assigned color.
func (a *Assigner) AddPrecoloredVertex(id string, color int) {
	a.g.AddVertex(id, &vertexState{color: color})
}
// AddUndirectedEdge creates an un-directed edge from srcID to dstID.
func (a *Assigner) AddUndirectedEdge(srcID, destID string) error {
	if err := a.g.AddEdge(srcID, destID, nil); err != nil {
		return err
	}

	// Create a reverse edge from the newly created edge as the source and the previous destination
	// as destination.
	return a.g.AddEdge(destID, srcID, nil)
}

// AssignColors executes the Jones/Plassmann algorithm on the graph and invokes
// the user-defined visitor function for each vertex in the graph.
func (a *Assigner) AssignColors(ctx context.Context, visitor func(vertexID string, color int)) (int, error) {
	exec := a.executorFactory(a.g, bspgraph.ExecutorCallbacks{
		ShouldPostStepKeepRunning: func(_ context.Context, _ *bspgraph.Graph, activeInStep int) (bool, error) {
			// Stop when all vertices have been colored.
			return activeInStep != 0, nil
		},
	})

	if err := exec.RunToCompletion(ctx); err != nil {
		return 0, nil
	}

	var numOfColors int
	for vertexID, v := range a.g.Vertices() {
		vState := v.Value().(*vertexState)
		if vState.color > numOfColors {
			numOfColors = vState.color
		}

		visitor(vertexID, vState.color)
	}

	return numOfColors, nil
}

func assignColorsToGraph(g *bspgraph.Graph, v *bspgraph.Vertex, msgIt message.Iterator) error {
	v.Freeze()
	vState := v.Value().(*vertexState)

	// Initialization. If this is an unconnected vertex without a color, assign the first possible color.
	if g.Superstep() == 0 {
		if vState.color == 0 && len(v.Edges()) == 0 {
			vState.color = 1

			return nil
		}

		vState.token = rand.Int()
		vState.usedColors = make(map[int]bool)

		return g.BroadcastToNeighbors(v, vState.asMessage(v.ID()))
	}

	if vState.color != 0 {
		return nil
	}

	// Process neighbor updates and update edge color assignments. Also,
	// figure out if we have the highest token number from un-colored
	// neighbors so we get to pick a color next.
	//
	// If our token is also assigned to a neighbor (highly unlikely) compare
	// the vertex IDs to break the tie.
	shouldPickNextColor := true
	myID := v.ID()

	for msgIt.Next() {
		msg := msgIt.Message().(*vertexStateMessage)
		if msg.Color != 0 {
			vState.usedColors[msg.Color] = true
		} else if vState.token < msg.Token || (vState.token == msg.Token) && myID < msg.ID {
			shouldPickNextColor = false
		}
	}

	// If it's not yet our turn to pick a color keep broadcasting our token
	// to each one of our neighbors.
	if !shouldPickNextColor {
		return g.BroadcastToNeighbors(v, vState.asMessage(myID))
	}

	// Find the minimum unused color, assign it to us and announce it to neighbors
	for nextColor := 1; ; nextColor++ {
		if vState.usedColors[nextColor] {
			continue
		}

		vState.color = nextColor

		return g.BroadcastToNeighbors(v, vState.asMessage(myID))
	}
}