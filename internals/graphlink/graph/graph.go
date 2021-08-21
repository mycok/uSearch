package graph

import (
	"time"

	"github.com/google/uuid"
)

// Graph is implemented by objects that mutate or query a link graph.
type Graph interface {
	// UpsertLink creates a new or updates an existing link 
	UpsertLink(link *Link) error

	// FindLink performs a lookup by id
	FindLink(id uuid.UUID) (*Link, error)

	// Links returns an alterator for a set of links whose id's belong
	// to the (fromID, toID) range and were retrieved before the (retrievedBefore) time
	Links(fromID, toID uuid.UUID, retrievedBefore time.Time) (LinkIterator, error)

	// UpsertEdge creates a new or updates an existing edge
	UpsertEdge(edge *Edge) error

	// RemoveStaleEdges removes any edge that originates from a specific link ID
	// and was updated before the specified (updatedBefore) time
	RemoveStaleEdges(fromID uuid.UUID, updatedBefore time.Time) error

	// Edges returns an iterator for a set of edges whose source vertex id's
	// belong to the (fromID, toID) range and were updated before the (updatedBefore) time
	Edges(fromID, toID uuid.UUID, updatedBefore time.Time) (EdgeIterator, error)
}

// Link encapsulates all data fields representing a link crawled by uSearch
type Link struct {
	// Unique identifier for the link
	ID uuid.UUID

	// The link target
	URL string

	// The timestamp when the link was last retrieved
	RetrievedAt time.Time
}

// Edge describes a graph edge that originates from Src and terminates
// at Dest.
type Edge struct {
	// Unique identifier for an Edge
	ID uuid.UUID

	// Unique identifier for the origin link
	Src uuid.UUID

	// Unique identifier for the destination link
	Dest uuid.UUID

	// The timestamp when the edge was last updated
	UpdatedAt time.Time
}

// LinkIterator is implemented by objects that iterate graph links
type LinkIterator interface {
	Iterator

	// Link returns the currently fetched link object
	Link() *Link
}

// EdgeIterator is implemented by objects that iterate graph edges
type EdgeIterator interface {
	Iterator

	// Edge returns the currently fetched Edge object
	Edge() *Edge
}

// Iterator is embedded & / or implemented by objects that require iteration functionality
type Iterator interface {
	// Next advances the iterator. If no more items are available or an
    // error occurs, calls to Next() return false.
	Next() bool

	// Error returns the last error encountered by the iterator
	Error() error

	// Close releases any resources linked to the iterator
	Close()
}

