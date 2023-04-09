/*
	graph package defines types that outline the behavior of link
	and edge or graph data stores.
*/

package graph

import (
	"time"

	"github.com/google/uuid"
)

// Graph should be implemented by graph data stores / types.
type Graph interface {
	// UpsertLink creates a new or updates an existing link.
	UpsertLink(link *Link) error

	// FindLink performs a link lookup by id.
	FindLink(id uuid.UUID) (*Link, error)

	// Links returns an iterator for a set of links whose id's belong
	// to the [fromID, toID] range and were retrieved before the [retrievedBefore]
	// time.
	Links(fromID, toID uuid.UUID, retrievedBefore time.Time) (LinkIterator, error)

	// UpsertEdge creates a new or updates an existing edge.
	UpsertEdge(edge *Edge) error

	// RemoveStaleEdges removes any edge that originates from a specific link ID
	// and was updated before the specified [updatedBefore] time.
	RemoveStaleEdges(fromID uuid.UUID, updatedBefore time.Time) error

	// Edges returns an iterator for a set of edges whose source vertex id's
	// belong to the [fromID, toID] range and were updated before the
	// [updatedBefore] time.
	Edges(fromID, toID uuid.UUID, updatedBefore time.Time) (EdgeIterator, error)
}

// LinkIterator is implemented by types that iterate graph links.
type LinkIterator interface {
	Iterator

	// Link returns the currently fetched link object.
	Link() *Link
}

// EdgeIterator is implemented by types that iterate graph edges.
type EdgeIterator interface {
	Iterator

	// Edge returns the currently fetched Edge object.
	Edge() *Edge
}

// Iterator should be embedded / implemented by types that require
// iteration functionality.
type Iterator interface {
	// Next loads the next item, returns false when no more documents
	// are available or when an error occurs.
	Next() bool

	// Error returns the last error encountered by the iterator.
	Error() error

	// Close releases any resources allocated to the iterator.
	Close() error
}

// Link represents a web link. it serves as a model / schema object.
type Link struct {
	ID          uuid.UUID // Link unique identifier
	URL         string    // Link target
	RetrievedAt time.Time // Last retrieved / crawled timestamp
}

// Edge represents a graph edge that originates from Src and terminates
// at Dest. it serves as a model / schema object.
type Edge struct {
	ID        uuid.UUID // Edge unique identifier
	Src       uuid.UUID // Unique identifier for edge origin / source link ID
	Dest      uuid.UUID // Unique identifier for edge destination / destination link ID
	UpdatedAt time.Time // Last updated timestamp
}
