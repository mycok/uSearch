package crawler

import (
	"net/http"
	"time"

	"github.com/mycok/uSearch/internal/graphlink/graph"
	"github.com/mycok/uSearch/internal/textindexer/index"

	"github.com/google/uuid"
)

// URLGetter is implemented by objects that can perform HTTP GET requests.
type URLGetter interface {
	Get(url string) (*http.Response, error)
}

// PrivateNetworkDetector is implemented by objects that can detect whether a
// host resolves to a private network address.
type PrivateNetworkDetector interface {
	IsPrivate(host string) (bool, error)
}

// MiniGraph is implemented by objects that can upsert links and edges into a link
// graph instance.
type MiniGraph interface {
	// UpsertLink creates a new or updates an existing link.
	UpsertLink(link *graph.Link) error

	// UpsertEdge creates a new or updates an existing edge.
	UpsertEdge(edge *graph.Edge) error

	// RemoveStaleEdges removes any edge that originates from a specific link ID
	// and was updated before the specified [updatedBefore] time.
	RemoveStaleEdges(fromID uuid.UUID, updatedBefore time.Time) error
}

// MiniIndexer is implemented by objects that can index documents
// discovered by the crawler component.
type MiniIndexer interface {
	// Index inserts a new document to the index or updates the index entry
	// for and existing document.
	Index(doc *index.Document) error
}

// Config encapsulates the configuration options for creating a new Crawler.
type Config struct {
	// A PrivateNetworkDetector instance.
	PrivateNetworkDetector PrivateNetworkDetector
	// A URLGetter instance for fetching links.
	URLGetter URLGetter
	// A GraphUpdater instance for up-serting new links to the link graph.
	Graph MiniGraph
	// A TextIndexer instance for indexing the content of each retrieved link.
	Indexer MiniIndexer
	// The number of concurrent workers used for retrieving links.
	NumOfFetchWorkers int
}
