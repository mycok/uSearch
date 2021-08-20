package graph

import (
	"time"

	"github.com/google/uuid"
)

// Graph is implemented by objects that mutate or query a link graph.
type Graph interface {
	UpsertLink(link *Link) error
	FindLink(id uuid.UUID) (*Link, error)

	UpsertEdge(edge *Edge) error
	RemoveStaleEdges(fromID uuid.UUID, updatedBefore time.Time) error

	Links(fromID, toID uuid.UUID, retrievedBefore time.Time) (LinkIterator, error)
	Edges(fromID, toID uuid.UUID, updatedBefore time.Time) (EdgeIterator, error)
}

// Link encapsulates all data fields representing a link crawled by uSearch
type Link struct {
	ID uuid.UUID
	URL string
	RetrievedAt time.Time
}

// Edge describes a graph edge that originates from Src and terminates
// at Dest.
type Edge struct {
	ID uuid.UUID
	Src uuid.UUID
	Dest uuid.UUID
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

