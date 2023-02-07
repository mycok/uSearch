package graphtest

import (
	"github.com/mycok/uSearch/linkgraph/graph"
)

// BaseSuite defines a set of re-usable graph-related tests that can
// be executed against any concrete type that implements the graph.Graph interface.
type BaseSuite struct {
	g graph.Graph
}

// SetGraph configures the test-suite to run all tests against an instance
// of graph.Graph.
func (s *BaseSuite) SetGraph(g graph.Graph) {
	s.g = g
}
