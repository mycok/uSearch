package memory

import (
	"fmt"

	"github.com/blevesearch/bleve"

	"github.com/mycok/uSearch/textindexer/index"
)

// Static and compile-time check to ensure docIterator implements
// index.Iterator interface.
var _ index.Iterator = (*docIterator)(nil)

// docIterator is an index.Iterator implementation for the in-memory index.
type docIterator struct {
	// Pointer to an inMemoryIndex instance.
	idx *InMemoryIndex
	// Search Request Object.
	searchReq *bleve.SearchRequest
	// Cumulative index tracks the absolute position in the result list.
	cumIdx uint64
	// Search Result Index tracks the position in the current page list.
	searchResIdx int
	// Search Result Object.
	searchRes *bleve.SearchResult
	// Application Document.
	doc *index.Document
	// Last error encountered by the iterator.
	lastErr error
}

// Next loads the next item, returns false when no more items
// are available or when an error occurs.
func (i *docIterator) Next() bool {
	if i.lastErr != nil || i.searchRes == nil || i.cumIdx >= i.searchRes.Total {
		return false
	}

	// Check if we need to fetch the next result batch and if so, update
	// the search request cursor and perform a new search.
	if i.searchResIdx >= i.searchRes.Hits.Len() {
		i.searchReq.From += i.searchReq.Size
		i.searchRes, i.lastErr = i.idx.idx.Search(i.searchReq)
		if i.lastErr != nil {
			return false
		}

		// Reset the search result index for the new page of results.
		i.searchResIdx = 0
	}

	// Generate the next ID to be used for doc lookup.
	nextID := i.searchRes.Hits[i.searchResIdx].ID
	doc, exists := i.idx.docs[nextID]
	if !exists {
		i.lastErr = fmt.Errorf("find by ID: %w", index.ErrNotFound)

		return false
	}

	i.doc = doc
	i.searchResIdx++
	i.cumIdx++

	return true
}

// Document returns the current document from the result set.
func (i *docIterator) Document() *index.Document {
	return i.doc
}

// TotalCount returns the approximated total number of search results.
func (i *docIterator) TotalCount() uint64 {
	if i.searchRes == nil {
		return 0
	}

	return i.searchRes.Total
}

// Error returns the last error encountered by the iterator.
func (i *docIterator) Error() error {
	return i.lastErr
}

// Close releases any resources allocated to the iterator.
func (i *docIterator) Close() error {
	i.idx = nil
	i.searchReq = nil

	if i.searchRes != nil {
		i.cumIdx = i.searchRes.Total
	}

	return nil
}
