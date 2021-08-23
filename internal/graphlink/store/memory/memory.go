/*
Package memory includes an in-memory link graph store concrete implementation
of the Graph, LinkIterator, EdgeIterator and Iterator interface.
*/
package memory

import (
	"fmt"
	"sync"
	"time"

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
// match is found.
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

// Links returns an iterator for the set of links whose IDs belong to the
// [fromID, toID) range and were retrieved before the provided timestamp.
func (s *InMemoryGraph) Links(fromID, toID uuid.UUID, retrievedBefore time.Time) (graph.LinkIterator, error) {
	from, to := fromID.String(), toID.String()

	s.mu.RLock()

	var list []*graph.Link
	for linkID, link := range s.links {
		if id := linkID.String(); id >= from && id < to && link.RetrievedAt.Before(retrievedBefore) {
			list = append(list, link)
		}
	}

	s.mu.RUnlock()

	return &LinkIterator{s: s, links: list}, nil
}

// UpsertEdge creates a new edge or updates an existing edge.
func (s *InMemoryGraph) UpsertEdge(edge *graph.Edge) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, srcExists := s.links[edge.Src]
	_, destExists := s.links[edge.Dest]

	if !srcExists || !destExists {
		return fmt.Errorf("upsert edge: %w", graph.ErrUnknownEdgeLinks)
	}

	// Loop through the edgelist that matches the provided edge.src to check whether
	// the provided edge already exists.
	for _, edgeID := range s.linkEdgeMap[edge.Src] {
		existingEdge := s.edges[edgeID]
		if existingEdge.Src == edge.Src && existingEdge.Dest == edge.Dest {
			existingEdge.UpdatedAt = time.Now()
			*edge = *existingEdge

			return nil
		}
	}

	// Attach an ID and upsert the new edge
	for {
		edge.ID = uuid.New()
		if s.edges[edge.ID] == nil {
			break
		}
	}

	edge.UpdatedAt = time.Now()
	eCopy := new(graph.Edge)
	*eCopy = *edge
	s.edges[eCopy.ID] = eCopy

	// Append the edge ID to the list of edges originating from the
	// edge's source link.
	s.linkEdgeMap[edge.Src] = append(s.linkEdgeMap[edge.Src], eCopy.ID)

	return nil
}
