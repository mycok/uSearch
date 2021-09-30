package bspgraph

// Edge represents a directed edge in the Graph.
type Edge struct {
	destID string
	value  interface{}
}

// DstID returns the vertex ID that corresponds to this edge's target endpoint.
func (e *Edge) DestID() string { return e.destID }

// Value returns the value associated with this edge.
func (e *Edge) Value() interface{} { return e.value }

// SetValue sets the provided value to the associated edge.
func (e *Edge) SetValue(val interface{}) { e.value = val }
