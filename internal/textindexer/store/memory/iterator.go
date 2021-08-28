package memory

import (
	"github.com/mycok/uSearch/internal/textindexer/index"

	"github.com/blevesearch/bleve"
)

// bleveIterator implements index.Iterator.
type bleveIterator struct {
	idx       *InMemoryBleveIndexer
	searchReq *bleve.SearchRequest
	cumIdx    uint64
	srIdx     int
	sr        *bleve.SearchResult
	doc       *index.Document
	lastErr   error
}

// Close implements index.Iterator.Close() and releases any allocated resources
// still being used by the iterator.
func (i *bleveIterator) Close() error {
	i.idx = nil
	i.searchReq = nil
	if i.sr != nil {
		i.cumIdx = i.sr.Total
	}

	return nil
}

// Next implements index.Iterator.Next(). loads the next document matching the search query.
// It returns false if no more documents are available.
func (i *bleveIterator) Next() bool {
	if i.lastErr != nil || i.sr == nil || i.cumIdx >= i.sr.Total {
		return false
	}

	// Check if we need to fetch the next result batch.
	if i.srIdx >= i.sr.Hits.Len() {
		i.searchReq.From += i.searchReq.Size
		if i.sr, i.lastErr = i.idx.idx.Search(i.searchReq); i.lastErr != nil {
			return false
		}

		i.srIdx = 0
	}

	// Generate the next ID to be used for doc lookup.
	nextID := i.sr.Hits[i.srIdx].ID
	if i.doc, i.lastErr = i.idx.findByID(nextID); i.lastErr != nil {
		return false
	}

	i.cumIdx++
	i.srIdx++

	return true
}

// Error implements index.Iterator.Error() and returns the last error
// returned by calling index.Iterator.Next().
func (i *bleveIterator) Error() error {
	return i.lastErr
}

// Document implements index.Iterator.Document() and returns the
// current document from the search result set.
func (i *bleveIterator) Document() *index.Document {
	return i.doc
}

// TotalCount implements index.Iterator.TotalCount() and returns
//  the approximate number of search results.
func (i *bleveIterator) TotalCount() uint64 {
	if i.sr == nil {
		return 0
	}

	return i.sr.Total
}
