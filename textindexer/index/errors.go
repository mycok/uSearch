package index

import "errors"

var (
	// ErrNotFound is returned by the indexer when it attempts to look up
	// a document that does not exist.
	ErrNotFound = errors.New("not found")

	// ErrMissingLinkID is returned when an indexer attempts to index a document
	// with an invalid / missing link ID.
	ErrMissingLinkID = errors.New("document has missing / invalid id")
)
