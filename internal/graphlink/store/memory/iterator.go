package memory

import "github.com/mycok/uSearch/internal/graphlink/graph"

// linkIterator type servers as a graph.LinkIterator interface concrete
// implementation for the in-memory graph store.
type linkIterator struct {
	s            *InMemoryGraph
	links        []*graph.Link
	currentIndex int
}

// Next implements graph.LinkIterator.Next()
func (i *linkIterator) Next() bool {
	if i.currentIndex >= len(i.links) {
		return false
	}

	i.currentIndex++

	return true
}

// Error implements graph.LinkIterator.Error()
func (i *linkIterator) Error() error {
	return nil
}

// Close implements graph.LinkIterator.Close()
func (i *linkIterator) Close() error {
	return nil
}

// Link implements graph.LinkIterator.Link() which returns a link document
// whenever a call to graph.LinkIterator.Next() returns true
func (i *linkIterator) Link() *graph.Link {
	// The link pointer contents may be overwritten by a graph update, to
	// avoid data-races we first acquire the read lock and then clone the link.
	i.s.mu.RLock()

	link := new(graph.Link)
	*link = *i.links[i.currentIndex-1]

	i.s.mu.RUnlock()

	return link
}

// edgeIterator is a graph.EdgeIterator interface concrete
// implementation for the in-memory graph store.
type edgeIterator struct {
	s            *InMemoryGraph
	edges        []*graph.Edge
	currentIndex int
}

// Next implements graph.EdgeIterator.Next() which returns true when
// there is a valid edge object to be returned and false otherwise.
func (i *edgeIterator) Next() bool {
	if i.currentIndex >= len(i.edges) {
		return false
	}

	i.currentIndex++

	return true
}

// Error implements graph.EdgeIterator.Error()
func (i *edgeIterator) Error() error {
	return nil
}

// Close implements graph.EdgeIterator.Close()
func (i *edgeIterator) Close() error {
	return nil
}

// Edge implements graph.EdgeIterator.Edge() which returns an edge object
// whenever a call to graph.EdgeIterator.Next() returns true.
func (i *edgeIterator) Edge() *graph.Edge {
	// The edge pointer contents may be overwritten by a graph update; to
	// avoid data-races we first acquire the read lock and then clone the edge
	i.s.mu.RLock()

	edge := new(graph.Edge)
	*edge = *i.edges[i.currentIndex-1]

	i.s.mu.RUnlock()

	return edge
}
