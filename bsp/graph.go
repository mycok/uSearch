/*
	bsp is a large scale graph processing implementation based on the bulk
	synchronization parallel (BSP model).
*/

package bsp

import (
	"errors"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/mycok/uSearch/bsp/queue"
)

var (
	// ErrUnknownEdgeSource is returned when the source vertex is not present
	// in the graph.
	ErrUnknownEdgeSource = errors.New("source vertex is not part of the graph")

	// ErrDestinationLocal is returned when a message destination is not a
	// valid remote destination / destination is local.
	ErrDestinationLocal = errors.New(
		"message destination is assigned to the local graph",
	)

	// ErrInvalidMessageDestination is returned when the message destination
	// cannot be resolved to any (local or remote) vertex.
	ErrInvalidMessageDestination = errors.New("invalid message destination")
)

// Graph provides graph processing functionality.
type Graph struct {
	wg                sync.WaitGroup
	superStep         int
	activeInStep      int64
	pendingInStep     int64
	aggregators       map[string]Aggregator
	vertices          map[string]*Vertex
	computeFn         ComputeFunc
	queueFactory      queue.Factory
	relayer           Relayer
	vertexChan        chan *Vertex
	errChan           chan error
	stepCompletedChan chan struct{}
}

// NewGraph creates a new Graph instance using the provided configuration. It
// is important for callers to invoke Close() on the returned graph instance
// when they are done using it.
func NewGraph(cfg GraphConfig) (*Graph, error) {
	// Validate the provided configuration.
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("graph config validation failed: %w", err)
	}

	g := &Graph{
		computeFn:    cfg.ComputeFn,
		queueFactory: cfg.QueueFactory,
		aggregators:  make(map[string]Aggregator),
		vertices:     make(map[string]*Vertex),
	}

	g.startWorkers(cfg.ComputeWorkers)

	return g, nil
}

// Close releases any resources associated with the graph.
func (g *Graph) Close() error {
	close(g.vertexChan)
	g.wg.Wait()

	return g.Reset()
}

// Reset the state of the graph by resetting the superStep counter,
// re-instantiating both the vertices and aggregators map objects
//
// Note: Reset operations are useful only when a client wants to re-use the
// same graph instance to run other new steps.
func (g *Graph) Reset() error {
	g.superStep = 0

	for _, v := range g.vertices {
		// Close the both input and output message queue's for each vertex.
		for i := 0; i < 2; i++ {
			if err := v.msgQueues[i].Close(); err != nil {
				return fmt.Errorf(
					"closing message queue %d for vertex %v failed", i, v,
				)
			}
		}
	}

	g.vertices = make(map[string]*Vertex)
	g.aggregators = make(map[string]Aggregator)

	return nil
}

// Vertices returns the graph vertex map.
func (g *Graph) Vertices() map[string]*Vertex {
	return g.vertices
}

// AddVertex adds a new vertex to the graph. If the vertex already exists,
// its value will be overwritten / updated instead.
func (g *Graph) AddVertex(id string, value interface{}) {
	v, exists := g.vertices[id]
	if !exists {
		v = &Vertex{
			id: id,
			msgQueues: [2]queue.Queue{
				g.queueFactory(),
				g.queueFactory(),
			},
			active: true,
		}

		g.vertices[id] = v
	}

	// Update the vertex value instead.
	v.SetValue(value)
}

// AddEdge adds a directed edge from source to destination and annotates it
// with the specified initValue. By design, edges are owned by the source
// vertices (destinations can be either local or remote) vertices / nodes and
// therefore srcID must resolve to a local vertex. Otherwise, AddEdge returns
// an error.
func (g *Graph) AddEdge(srcID, destID string, value interface{}) error {
	srcVertex, exists := g.vertices[srcID]
	if !exists {
		return fmt.Errorf(
			"create edge from %q to %q: %w", srcID, destID, ErrUnknownEdgeSource,
		)
	}

	srcVertex.edges = append(srcVertex.edges, &Edge{
		destID: destID,
		value:  value,
	})

	return nil
}

// RegisterAggregator adds an aggregator with the specified name into the graph.
func (g *Graph) RegisterAggregator(name string, aggr Aggregator) {
	g.aggregators[name] = aggr
}

// Aggregator returns the aggregator with the specified name or nil if the
// aggregator does not exist.
func (g *Graph) Aggregator(name string) Aggregator {
	return g.aggregators[name]
}

// Aggregators returns a map of all currently registered aggregators.
func (g *Graph) Aggregators() map[string]Aggregator {
	return g.aggregators
}

// RegisterRelayer configures a Relayer that the graph will invoke when
// attempting to deliver a message to a vertex that is not known locally but
// could potentially be owned by a remote graph instance.
func (g *Graph) RegisterRelayer(relayer Relayer) {
	g.relayer = relayer
}

// BroadcastToNeighbors is a helper function that broadcasts a single message
// to each neighbor(edge) of a particular vertex. Messages are queued for delivery
// and will be processed by recipients in the following superStep.
func (g *Graph) BroadcastToNeighbors(v *Vertex, msg queue.Message) error {
	for _, e := range v.edges {
		if err := g.SendMessage(e.destID, msg); err != nil {
			return err
		}
	}

	return nil
}

// SendMessage attempts to deliver a message to the vertex with the specified
// destination ID. Messages are queued for delivery and will be processed by
// recipients in the following superStep.
//
// If the destination ID is not known by this graph, it might still be a valid
// ID for a vertex that is owned by a remote graph instance. If the client
// defined a Relayer when configuring the graph, SendMessage will delegate
// message delivery to it.
//
// On the other hand, if no Relayer is defined or if the defined relayer returns
// an ErrDestinationIsLocal error, SendMessage will return an
// ErrInvalidMessageDestination to the caller.
func (g *Graph) SendMessage(destID string, msg queue.Message) error {
	// If the vertex is known to this local graph instance queue then
	// message directly so it can be delivered at the next superStep.
	destVertex, exists := g.vertices[destID]
	if exists {
		queueIdx := (g.superStep + 1) % 2

		return destVertex.msgQueues[queueIdx].Enqueue(msg)
	}

	// If the vertex is not known locally by this graph but might be known to a
	// partition that is processed by another node and if a remote relayer has
	// was configured, delegate the message sending operation to it.
	if g.relayer != nil {
		if err := g.relayer.Relay(destID, msg); !errors.Is(
			err, ErrDestinationLocal) {

			return err
		}
	}

	return fmt.Errorf(
		"message can't be delivered to %q: %w",
		destID, ErrInvalidMessageDestination,
	)
}

// SuperStep returns the current superStep value.
func (g *Graph) SuperStep() int {
	return g.superStep
}

// Step executes the next superStep and returns back the number of vertices
// that were processed either because they were still active or because they
// received a message.
//
// Note: A single step / superStep is only complete when all the available
// vertices are successfully processed or an error is returned by the compute
//
//	function.
func (g *Graph) step() (int, error) {
	g.activeInStep = 0
	g.pendingInStep = int64(len(g.vertices))

	if g.pendingInStep == 0 {
		return 0, nil
	}

	for _, v := range g.vertices {
		g.vertexChan <- v
	}

	// Block until the worker pool has finished processing all vertices.
	<-g.stepCompletedChan

	// Dequeue any errors.
	var err error

	select {
	case err = <-g.errChan:
	default:
	}

	return int(g.activeInStep), err
}

// startWorkers spins up numOfWorkers to execute each superStep.
func (g *Graph) startWorkers(numOfWorkers int) {
	g.vertexChan = make(chan *Vertex)
	// We use a buffered channel for errors because the step() method that
	// controls the step worker pool checks for errors at the end after ensuring
	// that all dispatched workers have finished their work. This eliminates the
	// possibility of dead locks that would be caused as a result of various
	// workers attempting to write an error to the error channel which already
	// contains an error which has not been read since error reading is supposed
	// to happen at the end.
	g.errChan = make(chan error, 1)
	g.stepCompletedChan = make(chan struct{})

	g.wg.Add(numOfWorkers)
	for i := 0; i < numOfWorkers; i++ {
		go g.stepWorker()
	}
}

// stepWorker polls the vertex channel for incoming vertices and executes the
// configured a compute function on each one. The worker will automatically exit
// when vertex channel is closed.
//
// Note: A step worker is run exactly once for each step / super step.
func (g *Graph) stepWorker() {
	for v := range g.vertexChan {
		queueIdx := g.superStep % 2
		if v.active || v.msgQueues[queueIdx].PendingMessages() {
			// Update the graph's activeInStep counter.
			_ = atomic.AddInt64(&g.activeInStep, 1)
			v.active = true
			// Invoke the compute function on the specific vertex.
			if err := g.computeFn(g, v, v.msgQueues[queueIdx].Messages()); err != nil {
				tryToEmitErr(g.errChan, fmt.Errorf(
					"running compute function for vertex %q failed: %w",
					v.ID(), err,
				))

			} else if err := v.msgQueues[queueIdx].DiscardMessages(); err != nil {
				tryToEmitErr(g.errChan, fmt.Errorf(
					"discarding unprocessed messages for vertex %q failed: %w",
					v.ID(), err,
				))
			}
		}

		// Check if this is the last vertex in the loop. This is done by
		// subtracting 1 from the g.pendingInStep value until it's value is 0.
		//
		// Note: we only write to the stepCompletedCh after processing the last vertex.
		if atomic.AddInt64(&g.pendingInStep, -1) == 0 {
			g.stepCompletedChan <- struct{}{}
		}
	}

	g.wg.Done()
}

func tryToEmitErr(errChan chan<- error, err error) {
	select {
	// Try to enqueue an error.
	case errChan <- err:
	// Error channel already contains another error that has not been read yet.
	default:
	}
}
