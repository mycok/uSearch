package bsp

// Edge represents a directed link / connection in a Graph. It links the source
// and destination vertex.
type Edge struct {
	// ID of the vertex that the edge points to.
	destID string
	// value represents the cost / weight attached to an edge based on specific
	// considerations and the context in which an edge is being used.
	value interface{}
}

// DestID returns the vertex ID that points to the edge's target endpoint.
func (e *Edge) DestID() string { return e.destID }

// Value returns the value associated with this edge.
func (e *Edge) Value() interface{} { return e.value }

// SetValue sets the provided value to the associated edge.
func (e *Edge) SetValue(val interface{}) { e.value = val }
