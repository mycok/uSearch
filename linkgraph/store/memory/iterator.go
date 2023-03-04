package memory

import "github.com/mycok/uSearch/linkgraph/graph"

// Static and compile-time check to ensure linkIterator implements
// graph.Iterator interface.
var _ graph.Iterator = (*linkIterator)(nil)

// linkIterator is a graph.LinkIterator implementation for the in-memory graph.
type linkIterator struct {
	// Pointer to an inMemoryGraph instance. it's used here to provide
	// access to the store's mutex object.
	store        *InMemoryGraph
	links        []*graph.Link
	currentIndex int
}

// Next loads the next item, returns false when no more links
// are available or when an error occurs.
func (i *linkIterator) Next() bool {
	if i.currentIndex >= len(i.links) {
		return false
	}

	i.currentIndex++

	return true
}

// Error returns the last error encountered by the iterator.
func (i *linkIterator) Error() error {
	return nil
}

// Close releases any resources allocated to the iterator.
func (i *linkIterator) Close() error {
	return nil
}

// Link returns the currently fetched link object.
func (i *linkIterator) Link() *graph.Link {
	// The link pointer contents may be overwritten by a graph update
	// outside this method. To avoid data-races, we acquire the read lock
	//  first and clone creating a local pointer to the queried link.
	i.store.mu.RLock()
	defer i.store.mu.RUnlock()

	l := new(graph.Link)
	*l = *i.links[i.currentIndex-1]

	return l
}

// edgeIterator is a graph.EdgeIterator implementation for the in-memory graph.
type edgeIterator struct {
	store        *InMemoryGraph // Provides access to the store mutex object.
	edges        []*graph.Edge
	currentIndex int
}

// Next advances the iterator. When no edges are available or when an
// error occurs, calls to Next() return false.
func (i *edgeIterator) Next() bool {
	if i.currentIndex >= len(i.edges) {
		return false
	}

	i.currentIndex++

	return true
}

// Error returns the last error recorded by the iterator.
func (i *edgeIterator) Error() error {
	return nil
}

// Close releases any resources linked to the iterator.
func (i *edgeIterator) Close() error {
	return nil
}

// Edge returns the currently fetched edge object.
func (i *edgeIterator) Edge() *graph.Edge {
	// The edge pointer contents may be overwritten by a graph update
	// outside this method. To avoid data-races, we acquire the read lock
	//  first and clone creating a local pointer to the queried edge.
	i.store.mu.RLock()
	defer i.store.mu.RUnlock()

	e := new(graph.Edge)
	*e = *i.edges[i.currentIndex-1]

	return e
}
