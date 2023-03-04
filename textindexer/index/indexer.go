package index

import "github.com/google/uuid"

// Indexer should implemented by objects that can index and search documents
// discovered by the crawler component.
type Indexer interface {
	// Index adds a new document or updates an existing index entry
	// in case of an existing document.
	Index(doc *Document) error

	// FindByID looks up a document by its link ID.
	FindByID(linkID uuid.UUID) (*Document, error)

	// Search performs a look up based on query and returns a result
	// iterator if successful or an error otherwise.
	Search(q Query) (Iterator, error)

	// UpdateScore updates the PageRank score for a document with the
	// specified link ID. If no such document exists, a placeholder
	// document with the provided score will be created.
	UpdateScore(linkID uuid.UUID, score float64) error
}

// Iterator should implemented by objects that can paginate search results.
type Iterator interface {
	// Next loads the next item, returns false when no more items
	// are available or when an error occurs.
	Next() bool

	// Error returns the last error encountered by the iterator.
	Error() error

	// Close releases any resources allocated to the iterator.
	Close() error

	// Document returns the current document from the result set.
	Document() *Document

	// TotalCount returns the approximated total number of search results.
	TotalCount() uint64
}

// QueryType represents an integer value for a specific query.
type QueryType uint8

const (
	// QueryTypeMatch queries for results that match parts
	// of the query expression.
	QueryTypeMatch QueryType = iota

	// QueryTypePhrase queries for results that exactly match the
	// entire query expression / phrase (full-text search).
	QueryTypePhrase
)

// Query defines properties for a search query.
type Query struct {
	// Defines how the indexer interprets the search expression.
	Type QueryType
	// Value to search for.
	Expression string
	// Determines the cursor value for the indexer / pagination.
	Offset uint64
}
