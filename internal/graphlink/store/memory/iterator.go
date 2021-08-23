package memory

import "github.com/mycok/uSearch/internal/graphlink/graph"

// LinkIterator is a graph.LinkIterator interface concrete implementation for
//  the in-memory graph store.
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
	// avoid data-races we acquire the read lock first and clone the link.
	i.s.mu.RLock()

	link := new(graph.Link)
	*link = *i.links[i.currentIndex-1]

	i.s.mu.RUnlock()

	return link
}
