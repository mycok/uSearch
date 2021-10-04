package bspgraph

import (
	"errors"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/mycok/uSearch/internal/bspgraph/message"
)

var (
	// ErrUnknownEdgeSource is returned by AddEdge when the source vertex
	// is not present in the graph.
	ErrUnknownEdgeSource = errors.New("source vertex is not part of the graph")

	// ErrDestinationIsLocal is returned by Relayer instances to indicate
	// that a message destination is actually owned by the local graph.
	ErrDestinationIsLocal = errors.New("message destination is assigned to the local graph")

	// ErrInvalidMessageDestination is returned by calls to SendMessage and
	// BroadcastToNeighbors when the destination cannot be resolved to any
	// (local or remote) vertex.
	ErrInvalidMessageDestination = errors.New("invalid message destination")
)

// Graph implements a parallel graph processor based on the concepts described
// in the Pregel paper.
type Graph struct {
	wg              sync.WaitGroup
	superstep       int
	aggregators     map[string]Aggregator
	vertices        map[string]*Vertex
	computeFn       ComputeFunc
	queueFactory    message.QueueFactory
	relayer         Relayer
	vertexCh        chan *Vertex
	errCh           chan error
	stepCompletedCh chan struct{}
	activeInStep    int64
	pendingInStep   int64
}

// NewGraph creates a new Graph instance using the specified configuration. It
// is important for callers to invoke Close() on the returned graph instance
// when they are done using it.
func NewGraph(cfg GraphConfig) (*Graph, error) {
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("graph config validation failed: %w", err)
	}

	g := &Graph{
		computeFn:    cfg.ComputeFunc,
		queueFactory: cfg.QueueFactory,
		aggregators:  make(map[string]Aggregator),
		vertices:     make(map[string]*Vertex),
	}

	g.startWorkers(cfg.ComputeWorkers)

	return g, nil
}

// Close releases any resources associated with the graph.
func (g *Graph) Close() error {
	close(g.vertexCh)
	g.wg.Wait()

	return g.Reset()
}

// Reset the state of the graph by removing any existing vertices or
// aggregators and resetting the superstep counter.
func (g *Graph) Reset() error {
	// Reset operations can be useful when a user wants to re-use the same graph to run other new steps.
	g.superstep = 0
	for _, v := range g.vertices {
		// Close the both input and output message queue's for each vertex.
		for i := 0; i < 2; i++ {
			if err := v.msgQueues[i].Close(); err != nil {
				return fmt.Errorf("closing message queue #%d for vertex %v failed: %w", i, v.ID(), err)
			}
		}
	}

	g.vertices = make(map[string]*Vertex)
	g.aggregators = make(map[string]Aggregator)

	return nil
}

// Vertices returns the graph vertices as a map where the key is the vertex ID.
func (g *Graph) Vertices() map[string]*Vertex { return g.vertices }

// AddVertex inserts a new vertex with the specified id and initial value into
// the graph. If the vertex already exists, AddVertex will just overwrite its
// value with the provided initValue.
func (g *Graph) AddVertex(id string, initValue interface{}) {
	v := g.vertices[id]
	// Create a new vertex if it doesn't already exist.
	if v == nil {
		v = &Vertex{
			id: id,
			msgQueues: [2]message.Queue{
				g.queueFactory(),
				g.queueFactory(),
			},
			active: true,
		}
		g.vertices[id] = v
	}

	// Update the existing vertex value if it already exists.
	v.SetValue(initValue)
}

// AddEdge inserts a directed edge from src to destination and annotates it
// with the specified initValue. By design, edges are owned by the source
// vertices (destinations can be either local or remote) and therefore srcID
// must resolve to a local vertex. Otherwise, AddEdge returns an error.
func (g *Graph) AddEdge(srcID, destID string, initValue interface{}) error {
	srcVertex := g.vertices[srcID]
	if srcVertex == nil {
		return fmt.Errorf("create edge from %q to %q failed: %w", srcID, destID, ErrUnknownEdgeSource)
	}

	srcVertex.edges = append(srcVertex.edges, &Edge{
		destID: destID,
		value:  initValue,
	})

	return nil
}

// RegisterAggregator adds an aggregator with the specified name into the graph.
func (g *Graph) RegisterAggregator(name string, aggr Aggregator) { g.aggregators[name] = aggr }

// Aggregator returns the aggregator with the specified name or nil if the
// aggregator does not exist.
func (g *Graph) Aggregator(name string) Aggregator { return g.aggregators[name] }

// Aggregators returns a map of all currently registered aggregators.
func (g *Graph) Aggregators() map[string]Aggregator { return g.aggregators }

// RegisterRelayer configures a Relayer that the graph will invoke when
// attempting to deliver a message to a vertex that is not known locally but
// could potentially be owned by a remote graph instance.
func (g *Graph) RegisterRelayer(relayer Relayer) { g.relayer = relayer }

// BroadcastToNeighbors is a helper function that broadcasts a single message
// to each neighbor(edge) of a particular vertex. Messages are queued for delivery
// and will be processed by recipients in the next superstep.
func (g *Graph) BroadcastToNeighbors(v *Vertex, msg message.Message) error {
	for _, e := range v.edges {
		if err := g.SendMessage(e.destID, msg); err != nil {
			return err
		}
	}

	return nil
}

// SendMessage attempts to deliver a message to the vertex with the specified
// destination ID. Messages are queued for delivery and will be processed by
// recipients in the next superstep.
//
// If the destination ID is not known by this graph, it might still be a valid
// ID for a vertex that is owned by a remote graph instance. If the client has
// provided a Relayer when configuring the graph, SendMessage will delegate
// message delivery to it.
//
// On the other hand, if no Relayer is defined or the configured
// RemoteMessageSender returns a ErrDestinationIsLocal error, SendMessage will
// first check whether an UnknownVertexHandler has been provided at
// configuration time and invoke it. Otherwise, an ErrInvalidMessageDestination
// is returned to the caller.
func (g *Graph) SendMessage(destID string, msg message.Message) error {
	// If the vertex is known to the local graph instance queue then
	// message directly so it can be delivered at the next superstep.
	destVertex := g.vertices[destID]
	if destVertex != nil {
		queueIndex := (g.superstep + 1) % 2

		return destVertex.msgQueues[queueIndex].Enqueue(msg)
	}

	// If the vertex is not known locally but might be known to a partition
	// that is processed at another node and if a remote relayer has been
	// configured delegate the message send operation to it.
	if g.relayer != nil {
		if err := g.relayer.Relay(destID, msg); !errors.Is(err, ErrDestinationIsLocal) {
			return err
		}
	}

	return fmt.Errorf("message cannot be delivered to %q: %w", destID, ErrInvalidMessageDestination)
}

// Superstep returns the current superstep value.
func (g *Graph) Superstep() int { return g.superstep }

// Step executes the next superstep and returns back the number of vertices
// that were processed either because they were still active or because they
// received a message.
func (g *Graph) step() (int, error) {
	g.activeInStep = 0
	g.pendingInStep = int64(len(g.vertices))

	// No work required.
	if g.pendingInStep == 0 {
		return 0, nil
	}

	for _, v := range g.vertices {
		g.vertexCh <- v
	}

	// Block until worker pool has finished processing all vertices.
	// This happens when a worker processes the last vertex from g.vertices and
	// an empty struct value is written to the stepCompletedCh.
	<-g.stepCompletedCh

	// Dequeque any errors.
	var err error
	select {
	case err = <-g.errCh: // try to dequeue an error, if no error is available run the default case instead.
	default: // no error available
	}

	return int(g.activeInStep), err
}

// startWorkers allocates the required channels and spins up numOfWorkers to
// execute each superstep.
func (g *Graph) startWorkers(numOfWorkers int) {
	g.vertexCh = make(chan *Vertex)
	g.errCh = make(chan error, 1)
	g.stepCompletedCh = make(chan struct{})

	g.wg.Add(numOfWorkers)
	for i := 0; i < numOfWorkers; i++ {
		go g.stepWorker()
	}
}

// stepWorker polls vertexCh for incoming vertices and executes the configured
// ComputeFunc for each one. The worker automatically exits when vertexCh gets
// closed.
func (g *Graph) stepWorker() {
	for v := range g.vertexCh {
		buffer := g.superstep % 2
		if v.active || v.msgQueues[buffer].PendingMessages() {
			_ = atomic.AddInt64(&g.activeInStep, 1)
			v.active = true

			if err := g.computeFn(g, v, v.msgQueues[buffer].Messages()); err != nil {
				tryToEmitErr(g.errCh, fmt.Errorf("running compute function for vertex %q failed: %w", v.ID(), err))
			} else if err := v.msgQueues[buffer].DiscardMessages(); err != nil {
				tryToEmitErr(g.errCh, fmt.Errorf("discarding unprocessed messages for vertex %q failed: %w", v.ID(), err))
			}
		}

		// Check if this is the last vertex in the loop.
		// Note: we only write to the stepCompletedCh after processing the last vertex.
		if atomic.AddInt64(&g.pendingInStep, -1) == 0 {
			g.stepCompletedCh <- struct{}{}
		}
	}

	g.wg.Done()
}

func tryToEmitErr(errChan chan<- error, err error) {
	select {
	case errChan <- err: //queue an error
	default: // channel already contains another error and has not been read yet
	}
}
