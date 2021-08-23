/*
Package memory includes an in-memory link graph store concrete implementation
of the Graph, LinkIterator, EdgeIterator and Iterator interface.
*/
package memory

import (
	"fmt"
	"sync"

	// "time"

	"github.com/mycok/uSearch/internal/graphlink/graph"

	"github.com/google/uuid"
)

// Compile-time check for ensuring InMemoryGraph implements Graph.
// var _ graph.Graph = (*InMemoryGraph)(nil)

// edgeList contains the slice of edge UUIDs that originate from a link in the graph.
type edgeList []uuid.UUID

// InMemoryGraph implements an in-memory link graph that can be concurrently
// accessed by multiple clients.
type InMemoryGraph struct {
	mu           sync.RWMutex
	links        map[uuid.UUID]*graph.Link
	edges        map[uuid.UUID]*graph.Edge
	linkURLIndex map[string]*graph.Link
	linkEdgeMap  map[uuid.UUID]edgeList
}

// NewInMemoryGraph creates a new in-memory link graph.
func NewInMemoryGraph() *InMemoryGraph {
	return &InMemoryGraph{
		links:        make(map[uuid.UUID]*graph.Link),
		edges:        make(map[uuid.UUID]*graph.Edge),
		linkURLIndex: make(map[string]*graph.Link),
		linkEdgeMap:  make(map[uuid.UUID]edgeList),
	}
}

// UpsertLink creates a new link or updates an existing link.
func (s *InMemoryGraph) UpsertLink(link *graph.Link) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if a link with the same URL already exists. If so, convert
	// this into an update and point the link ID to the existing link.
	if existing := s.linkURLIndex[link.URL]; existing != nil {
		link.ID = existing.ID
		originalTs := existing.RetrievedAt
		*existing = *link

		if originalTs.After(existing.RetrievedAt) {
			existing.RetrievedAt = originalTs
		}

		return nil
	}

	// Assign new ID and insert link
	for {
		link.ID = uuid.New()
		if s.links[link.ID] == nil {
			break
		}
	}

	lcopy := new(graph.Link)
	*lcopy = *link

	s.links[lcopy.ID] = lcopy
	s.linkURLIndex[lcopy.URL] = lcopy

	return nil
}

// FindLink performs a Link lookup by ID and returns an error if no
// match is found
func (s *InMemoryGraph) FindLink(id uuid.UUID) (*graph.Link, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	link := s.links[id]
	if link == nil {
		return nil, fmt.Errorf("find link: %w", graph.ErrNotFound)
	}

	lcopy := new(graph.Link)
	*lcopy = *link

	return lcopy, nil
}
