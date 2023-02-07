package memory

import (
	"fmt"
	"sync"
	"time"

	"github.com/blevesearch/bleve"
	"github.com/blevesearch/bleve/search/query"
	"github.com/google/uuid"

	"github.com/mycok/uSearch/textindexer/index"
)

// Size of each page of results that is cached locally by the iterator.
const batchSize = 10

// Static and compile-time check to ensure InMemoryIndex implements Indexer.
var _ index.Indexer = (*InMemoryIndex)(nil)

type bleveDoc struct {
	Title    string
	Content  string
	PageRank float64
}

// InMemoryIndex is an Indexer implementation that uses a bleve instance
// to index / catalogue and search documents but saves it's index in memory.
type InMemoryIndex struct {
	mu   sync.RWMutex // Mutex instance.
	docs map[string]*index.Document
	idx  bleve.Index // Pointer to an InMemoryIndex instance.
}

// NewInMemoryIndex instantiates and returns a text indexer that
// uses an in-memory bleve instance to index documents.
func NewInMemoryIndex() (*InMemoryIndex, error) {
	mapping := bleve.NewIndexMapping()
	idx, err := bleve.NewMemOnly(mapping)
	if err != nil {
		return nil, err
	}

	return &InMemoryIndex{
		idx:  idx,
		docs: make(map[string]*index.Document),
	}, nil
}

// Close releases / frees any previously allocated resources.
func (s *InMemoryIndex) Close() error {
	return s.idx.Close()
}

// Index adds a new document or updates an existing index entry
// in case of an existing document.
func (s *InMemoryIndex) Index(doc *index.Document) error {
	if doc.LinkID == uuid.Nil {
		return fmt.Errorf("index: %w", index.ErrMissingLinkID)
	}

	doc.IndexedAt = time.Now()
	dCopy := copyDoc(doc)
	key := dCopy.LinkID.String()

	// Acquire a general lock to avoid data races while mutating graph data.
	// Note: No other writes and reads are allowed for as long as this lock
	// is active.
	s.mu.Lock()
	defer s.mu.Unlock()
	// If updating, preserve existing page-rank score.
	if existingDoc, exists := s.docs[key]; exists {
		dCopy.PageRank = existingDoc.PageRank
	}

	if err := s.idx.Index(key, makeBleveDoc(dCopy)); err != nil {
		return fmt.Errorf("insert: %w", err)
	}

	s.docs[key] = dCopy

	return nil
}

// FindByID looks up a document by its link ID.
func (s *InMemoryIndex) FindByID(linkID uuid.UUID) (*index.Document, error) {
	// Read lock allows other processes or goroutines to perform reads by
	// concurrently acquiring other read locks.
	s.mu.RLock()
	defer s.mu.RUnlock()

	if doc, exists := s.docs[linkID.String()]; exists {
		return copyDoc(doc), nil
	}

	return nil, fmt.Errorf("find by ID: %w", index.ErrNotFound)
}

// Search performs a look up based on query and returns a result
// iterator if successful or an error otherwise.
func (s *InMemoryIndex) Search(q index.Query) (index.Iterator, error) {
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
	searchReq.From = int(q.Offset) // Initial result cursor point. it's always 0.

	sr, err := s.idx.Search(searchReq)
	if err != nil {
		return nil, fmt.Errorf("search: %w", err)
	}

	return &docIterator{
		idx:       s,
		searchReq: searchReq,
		searchRes: sr,
		cumIdx:    q.Offset,
	}, nil
}

// UpdateScore updates the PageRank score for a document with the
// specified link ID. If no such document exists, a placeholder
// document with the provided score will be created.
func (s *InMemoryIndex) UpdateScore(linkID uuid.UUID, score float64) error {
	// Acquire a general lock to avoid data races while mutating graph data.
	// Note: No other writes and reads are allowed for as long as this lock
	// is active.
	s.mu.Lock()
	defer s.mu.Unlock()

	key := linkID.String()
	doc, exists := s.docs[key]
	if !exists {
		doc = &index.Document{
			LinkID: linkID,
		}

		s.docs[key] = doc
	}

	doc.PageRank = score

	// When we update the page-rank / score of a index document, we must
	// ensure that we re-index in order to refresh / update the index. Failure
	// to do so would result into outdated search results.
	if err := s.idx.Index(key, makeBleveDoc(doc)); err != nil {
		return fmt.Errorf("update score: %w", err)
	}

	return nil
}

func copyDoc(doc *index.Document) *index.Document {
	dCopy := new(index.Document)
	*dCopy = *doc

	return dCopy
}

func makeBleveDoc(doc *index.Document) bleveDoc {
	return bleveDoc{
		Title:    doc.Title,
		Content:  doc.Content,
		PageRank: doc.PageRank,
	}
}
