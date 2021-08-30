package memory

import (
	"fmt"
	"sync"
	"time"

	"github.com/mycok/uSearch/internal/textindexer/index"

	"github.com/blevesearch/bleve"
	"github.com/blevesearch/bleve/search/query"

	"github.com/google/uuid"
)

// Size of each page of results that is cached locally by the iterator.
const batchSize = 10

// Compile-time check to ensure InMemoryBleveIndexer implements Indexer.
var _ index.Indexer = (*InMemoryBleveIndexer)(nil)

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
// uses an InMemoryBleve instance for indexing documents.
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

// FindByID looks up a document by its link ID.
func (i *InMemoryBleveIndexer) FindByID(linkID uuid.UUID) (*index.Document, error) {
	return i.findByID(linkID.String())
}

// Search the index for a particular query and return a result iterator.
func (i *InMemoryBleveIndexer) Search(q index.Query) (index.Iterator, error) {
	var bleveQuery query.Query

	switch q.Type {
	case index.QueryTypePhrase:
		bleveQuery = bleve.NewMatchPhraseQuery(q.Expression)
	default:
		bleveQuery = bleve.NewMatchQuery(q.Expression)
	}

	searchReq := bleve.NewSearchRequest(bleveQuery)
	searchReq.SortBy([]string{"-PageRank", "-_score"})
	searchReq.Size = batchSize
	searchReq.From = int(q.Offset)

	sr, err := i.idx.Search(searchReq)
	if err != nil {
		return nil, fmt.Errorf("search: %w", err)
	}

	return &bleveIterator{idx: i, searchReq: searchReq, sr: sr, cumIdx: q.Offset}, nil
}

// UpdateScore updates the PageRank score for a document with the specified
// link ID. If no such document exists, a placeholder document with the
// provided score will be created.
func (i *InMemoryBleveIndexer) UpdateScore(linkID uuid.UUID, score float64) error {
	i.mu.Lock()
	defer i.mu.Unlock()

	key := linkID.String()
	doc, found := i.docs[key]
	if !found {
		doc = &index.Document{
			LinKID: linkID,
		}
		i.docs[key] = doc

	}

	// Update the document's PageRank with the provided score and re-index the document.
	doc.PageRank = score
	if err := i.idx.Index(key, makeBleveDoc(doc)); err != nil {
		return fmt.Errorf("update score: %w", err)
	}

	return nil
}

// findByID looks up a document by its link UUID expressed as a string.
func (i *InMemoryBleveIndexer) findByID(linkID string) (*index.Document, error) {
	i.mu.Lock()
	defer i.mu.Unlock()

	if doc, found := i.docs[linkID]; found {
		return copyDoc(doc), nil
	}

	return nil, fmt.Errorf("find by id: %w", index.ErrNotFound)
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

