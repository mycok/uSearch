package index

import (
	"time"

	"github.com/google/uuid"
)

// Document describes a web-page whose content has been indexed.
type Document struct {
	// The ID of the graphlink entry that points to this document.
	LinKID uuid.UUID

	// The URL pointing to were the document was obtained from.
	URL string

	// The document title (if available).
	Title string

	// The document body
	Content string

	// The last time this document was indexed.
	IndexedAt time.Time

	// The PageRank score assigned to this document.
	PageRank float64
}
