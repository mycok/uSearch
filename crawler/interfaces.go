package crawler

import (
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/mycok/uSearch/linkgraph/graph"
	"github.com/mycok/uSearch/textindexer/index"
)

// URLGetter should be implemented by objects that perform
// HTTP GET requests to fetch link data.
type URLGetter interface {
	Get(url string) (*http.Response, error)
}

// PrivateNetworkDetector should be implemented by objects that can detect
// whether a host resolves to a private network address.
type PrivateNetworkDetector interface {
	IsNetworkPrivate(address string) (bool, error)
}

// MiniGraph should be implemented by objects that can upsert links and edges
// into a link graph instance.
type MiniGraph interface {
	// UpsertLink creates a new or updates an existing link.
	UpsertLink(link *graph.Link) error

	// UpsertEdge creates a new or updates an existing edge.
	UpsertEdge(edge *graph.Edge) error

	// RemoveStaleEdges removes any edge that originates from a specific link ID
	// and was updated before the specified [updatedBefore] time.
	RemoveStaleEdges(fromID uuid.UUID, updatedBefore time.Time) error
}

// MiniIndexer should be implemented by objects that can index documents
// discovered by the crawler component.
type MiniIndexer interface {
	// Index adds a new document or updates an existing index entry
	// in case of an existing document.
	Index(doc *index.Document) error
}
