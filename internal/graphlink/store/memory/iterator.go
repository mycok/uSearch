package memory

import "github.com/mycok/uSearch/internal/graphlink/graph"

// LinkIterator servers as a graph.LinkIterator interface concrete
// implementation for the in-memory graph store.
type LinkIterator struct {
	s            *InMemoryGraph
	links        []*graph.Link
	currentIndex int
}

// Next implements graph.LinkIterator.Next()
func (i *LinkIterator) Next() bool {
	if i.currentIndex >= len(i.links) {
		return false
	}

	i.currentIndex++

	return true
}

// Error implements graph.LinkIterator.Error()
func (i *LinkIterator) Error() error {
	return nil
}

// Close implements graph.LinkIterator.Close()
func (i *LinkIterator) Close() error {
	return nil
}

// Link implements graph.LinkIterator.Link()
func (i *LinkIterator) Link() *graph.Link {
	// The link pointer contents may be overwritten by a graph update, to
	// avoid data-races we first acquire the read lock and then clone the link.
	i.s.mu.RLock()

	link := new(graph.Link)
	*link = *i.links[i.currentIndex-1]

	i.s.mu.RUnlock()

	return link
}

// EdgeIterator ervers as a graph.LinkIterator interface concrete
// implementation for the in-memory graph store.
type EdgeIterator struct {
	s            *InMemoryGraph
	edges        []*graph.Edge
	currentIndex int
}

// Next implements graph.EdgeIterator.Next()
func (i *EdgeIterator) Next() bool {
	if i.currentIndex >= len(i.edges) {
		return false
	}

	i.currentIndex++

	return true
}

// Error implements graph.EdgeIterator.Error()
func (i *EdgeIterator) Error() error {
	return nil
}

// Close implements graph.EdgeIterator.Close()
func (i *EdgeIterator) Close() error {
	return nil
}

// Edge implements graph.EdgeIterator.Edge()
func (i *EdgeIterator) Edge() *graph.Edge {
	// The edge pointer contents may be overwritten by a graph update; to
	// avoid data-races we first acquire the read lock and then clone the edge
	i.s.mu.RLock()

	edge := new(graph.Edge)
	*edge = *i.edges[i.currentIndex-1]

	i.s.mu.RUnlock()

	return edge
}
