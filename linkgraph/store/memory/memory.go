package memory

import (
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/mycok/uSearch/linkgraph/graph"
)

// Static and compile-time check to ensure InMemoryGraph implements
// Graph interface.
var _ graph.Graph = (*InMemoryGraph)(nil)

// edgeList contains the slice of edge UUIDs that originate from a link in the
// graph.
type edgeList []uuid.UUID

// InMemoryGraph implements an in-memory link and edge graph that can be concurrently
// accessed by multiple clients.
type InMemoryGraph struct {
	mu            sync.RWMutex
	links         map[uuid.UUID]*graph.Link
	edges         map[uuid.UUID]*graph.Edge
	linkURLIndex  map[string]*graph.Link
	linkToEdgeMap map[uuid.UUID]edgeList // Maps links to edges originating from them.
}

// NewInMemoryGraph creates a new in-memory link graph.
func NewInMemoryGraph() *InMemoryGraph {
	return &InMemoryGraph{
		links:         make(map[uuid.UUID]*graph.Link),
		edges:         make(map[uuid.UUID]*graph.Edge),
		linkURLIndex:  make(map[string]*graph.Link),
		linkToEdgeMap: make(map[uuid.UUID]edgeList),
	}
}

// UpsertLink creates a new or updates an existing link.
func (s *InMemoryGraph) UpsertLink(link *graph.Link) error {
	// Acquire a general lock to avoid data races while mutating graph data.
	// Note: No other writes and reads are allowed for as long as this lock
	// is active.
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if a link with the same URL already exists.
	// If so, and the retrievedAt time of the provided link is more current
	//  than the existing link, convert the operation into an update, replace
	//  the provided link ID with the existing link ID and replace the content
	//  of the existing link with the content of the provided link.
	// else if the retrievedAt time is older or as old as the existing link,
	// then do nothing.
	if l, exists := s.linkURLIndex[link.URL]; exists {
		link.ID = l.ID
		existingLinkRetrievedAt := l.RetrievedAt
		*l = *link

		if existingLinkRetrievedAt.After(link.RetrievedAt) {
			l.RetrievedAt = existingLinkRetrievedAt
		}

		return nil
	}

	// Try to assign a random ID to a new link. in case the generated ID
	// is already used, run the ID generator until a unique ID is found.
	for {
		link.ID = uuid.New()
		if _, exists := s.links[link.ID]; !exists {
			break
		}
	}

	// Make a new local pointer to the link provided by the user.
	// This step protects the local link data from side-effects triggered
	// outside this method.
	lCopy := new(graph.Link)
	*lCopy = *link

	s.links[lCopy.ID] = lCopy
	s.linkURLIndex[lCopy.URL] = lCopy

	return nil
}

// FindLink performs a link lookup by id.
func (s *InMemoryGraph) FindLink(id uuid.UUID) (*graph.Link, error) {
	// Read lock allows other processes or goroutines to perform reads by
	// concurrently acquiring other read locks.
	s.mu.RLock()
	defer s.mu.RUnlock()

	l, exists := s.links[id]
	if !exists {
		return nil, fmt.Errorf("find link: %w", graph.ErrNotFound)
	}

	// Make a new local pointer to the link provided by the user.
	// This step protects the local link data from side-effects triggered
	// outside this method.
	lCopy := new(graph.Link)
	*lCopy = *l

	return lCopy, nil
}

// Links returns an iterator for a set of links whose id's belong
// to the [fromID, toID] range and were retrieved before the [retrievedBefore]
// time.
func (s *InMemoryGraph) Links(
	fromID, toID uuid.UUID, retrievedBefore time.Time,
) (graph.LinkIterator, error) {

	from := fromID.String()
	to := toID.String()

	// Read lock allows other processes or goroutines to perform reads by
	// concurrently acquiring other read locks.
	s.mu.RLock()
	defer s.mu.RUnlock()

	var list []*graph.Link
	for id, link := range s.links {
		idString := id.String()
		if idString >= from && idString < to && link.RetrievedAt.Before(retrievedBefore) {
			list = append(list, link)
		}
	}

	return &linkIterator{store: s, links: list}, nil
}

// UpsertEdge creates a new or updates an existing edge.
func (s *InMemoryGraph) UpsertEdge(edge *graph.Edge) error {
	// Acquire a general lock to avoid data races while mutating graph data.
	// Note: No other writes and reads are allowed for as long as this lock
	// is active.
	s.mu.Lock()
	defer s.mu.Unlock()

	_, isSrcExists := s.links[edge.Src]
	_, isDestExists := s.links[edge.Dest]
	if !isSrcExists || !isDestExists {
		return fmt.Errorf("upsert edge: %w", graph.ErrUnknownEdgeLinks)
	}

	// Loop through the edge-list for edges that belong to provided edge.src.
	for _, edgeID := range s.linkToEdgeMap[edge.Src] {
		existingEdge := s.edges[edgeID]
		// Compare UUID bits as opposed to converting them into strings.
		// If the provided edge already exists, simply update it's updatedAt
		// field.
		if existingEdge.Src == edge.Src && existingEdge.Dest == edge.Dest {
			existingEdge.UpdatedAt = time.Now()
			// Copy the updated contents of the matching edge to the provided
			// edge. ie: the provided edge now has the ID from the existing edge.

			// Note: This copy behavior is to be used in testing. The updated edge
			// pointer is used in test assertions
			*edge = *existingEdge

			return nil
		}
	}

	// Try to assign a random ID to a new edge. in case the generated ID
	// is already used, run the ID generator until a unique ID is found.
	for {
		edge.ID = uuid.New()
		if _, exists := s.edges[edge.ID]; !exists {
			break
		}
	}

	edge.UpdatedAt = time.Now()
	eCopy := new(graph.Edge)
	*eCopy = *edge

	s.edges[eCopy.ID] = eCopy
	s.linkToEdgeMap[eCopy.Src] = append(s.linkToEdgeMap[eCopy.Src], eCopy.ID)

	return nil
}

// RemoveStaleEdges removes any edge that originates from a specific link ID
// and was updated before the specified [updatedBefore] time.
func (s *InMemoryGraph) RemoveStaleEdges(
	fromID uuid.UUID, updatedBefore time.Time,
) error {

	// Acquire a general lock to avoid data races while mutating graph data.
	// Note: No other writes and reads are allowed for as long as this lock
	// is active.
	s.mu.Lock()
	defer s.mu.Unlock()

	var newEdgeList edgeList

	for _, id := range s.linkToEdgeMap[fromID] {
		edge := s.edges[id]
		if edge.UpdatedAt.Before(updatedBefore) {
			delete(s.edges, id)

			continue
		}

		newEdgeList = append(newEdgeList, id)
	}

	// Replace the old edge list for the link with the new edge list.
	s.linkToEdgeMap[fromID] = newEdgeList

	return nil
}

// Edges returns an iterator for a set of edges whose source vertex id's
// belong to the [fromID, toID] range and were updated before the
//
//	[updatedBefore] time.
func (s *InMemoryGraph) Edges(
	fromID, toID uuid.UUID, updatedBefore time.Time,
) (graph.EdgeIterator, error) {

	from := fromID.String()
	to := toID.String()

	// Read lock allows other processes or goroutines to perform reads by
	// concurrently acquiring other read locks.
	s.mu.RLock()
	defer s.mu.RUnlock()

	var list []*graph.Edge

	for id := range s.links {
		linkID := id.String()
		if linkID < from || linkID >= to {
			continue
		}

		for _, edgeID := range s.linkToEdgeMap[id] {
			if edge := s.edges[edgeID]; edge.UpdatedAt.Before(updatedBefore) {
				list = append(list, edge)
			}
		}
	}

	return &edgeIterator{store: s, edges: list}, nil
}
