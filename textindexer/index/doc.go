package index

import (
	"time"

	"github.com/google/uuid"
)

// Document defines a web-page whose content has been successfully indexed.
type Document struct {
	// ID of the link entry that points to this document.
	LinkID uuid.UUID

	// URL pointing to the source of the document content.
	URL string

	// Title of the document (if available).
	Title string

	// Body of the document.
	Content string

	// PageRank score assigned to this document.
	PageRank float64

	// Last time the document was indexed.
	IndexedAt time.Time
}
