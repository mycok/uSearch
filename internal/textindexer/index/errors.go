package index

import "errors"

var (
	// ErrNotFound is returned by the indexer when attempting to look up
	// a document that does not exist.
	ErrNotFound = errors.New("not found")

	// ErrMissingLinkID is returned when attempting to index a document
	// that does not specify a valid link ID.
	ErrMissingLinkID = errors.New("document does not provide a valid Link ID")
)
