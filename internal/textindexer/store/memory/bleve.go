package memory

import (
	"fmt"
	"sync"
	"time"

	"github.com/mycok/uSearch/internal/textindexer/index"

	"github.com/blevesearch/bleve"
	// "github.com/blevesearch/bleve/search/query"
	"github.com/google/uuid"
)

// Size of each page of results that is cached locally by the iterator.
// const batchSize = 10

// Compile-time check to ensure InMemoryBleveIndexer implements Indexer.
// var _ index.Indexer = (*InMemoryBleveIndexer)(nil)

type bleveDoc struct {
	Title    string
	Content  string
	PageRank float64
}

// InMemoryBleveIndexer is an Indexer implementation that uses an in-memory
// bleve instance to index / catalogue and search documents.
type InMemoryBleveIndexer struct {
	mu   sync.Mutex
	docs map[string]*index.Document
	idx  bleve.Index
}

// NewInMemoryBleveIndexer creates and returns a text indexer that
// uses an in-memorybleve instance for indexing documents.
func NewInMemoryBleveIndexer() (*InMemoryBleveIndexer, error) {
	mapping := bleve.NewIndexMapping()
	idx, err := bleve.NewMemOnly(mapping)
	if err != nil {
		return nil, err
	}

	return &InMemoryBleveIndexer{
		idx:  idx,
		docs: make(map[string]*index.Document),
	}, nil
}

// Close the indexer and release any allocated resources.
func (i *InMemoryBleveIndexer) Close() error {
	return i.idx.Close()
}

// Index inserts a new document to the index or updates the index entry
// for and existing document.
func (i *InMemoryBleveIndexer) Index(doc *index.Document) error {
	if doc.LinKID == uuid.Nil {
		return fmt.Errorf("index: %w", index.ErrMissingLinkID)
	}

	doc.IndexedAt = time.Now()
	dcopy := copyDoc(doc)
	key := dcopy.LinKID.String()

	i.mu.Lock()
	// If updating, preserve existing PageRank score
	if orig, exists := i.docs[key]; exists {
		dcopy.PageRank = orig.PageRank
	}

	if err := i.idx.Index(key, makeBleveDoc(dcopy)); err != nil {
		return fmt.Errorf("index: %w", err)
	}

	i.docs[key] = dcopy
	i.mu.Unlock()

	return nil
}

func makeBleveDoc(doc *index.Document) bleveDoc {
	return bleveDoc{
		Title:    doc.Title,
		Content:  doc.Content,
		PageRank: doc.PageRank,
	}
}

func copyDoc(doc *index.Document) *index.Document {
	docCopy := new(index.Document)
	*docCopy = *doc

	return docCopy
}
