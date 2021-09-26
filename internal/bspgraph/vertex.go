package bspgraph

import (
	"github.com/mycok/uSearch/internal/bspgraph/message"
)

// Vertex represents a vertex instance in the Graph.
type Vertex struct {
	id        string
	value     interface{}
	active    bool
	msgQueues [2]message.Queue
	edges     []*Edge
}

// ID returns the Vertex ID.
func (v *Vertex) ID() string { return v.id }

// Edges returns the list of outgoing edges from this vertex.
func (v *Vertex) Edges() []*Edge { return v.edges }

// Freeze marks the vertex as inactive. Inactive vertices will not be processed
// in the following supersteps unless they receive a message in which case they
// will be re-activated.
func (v *Vertex) Freeze() { v.active = false }

// Value returns the value associated with this vertex.
func (v *Vertex) Value() interface{} { return v.value }

// SetValue sets the provided value  to the associated vertex.
func (v *Vertex) SetValue(val interface{}) { v.value = val }
