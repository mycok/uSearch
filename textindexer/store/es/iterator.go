package es

import (
	"github.com/elastic/go-elasticsearch/v8"

	"github.com/mycok/uSearch/textindexer/index"
)

// Static and compile-time check to ensure esIterator implements
// index.Iterator interface.
var _ index.Iterator = (*esIterator)(nil)

// esIterator is an index.Iterator implementation for an elasticsearch index.
type esIterator struct {
	// Pointer to an esIterator instance.
	client *elasticsearch.Client
	// Search Request Object.
	searchReq map[string]interface{}
	// Cumulative index tracks the absolute position in the result list.
	cumIdx uint64
	// Search Result Index tracks the position in the current page list.
	searchResIdx int
	// Search Result Object.
	searchRes *esSearchRes
	// Application Document.
	doc *index.Document
	// Last error encountered by the iterator.
	lastErr error
}

// Next loads the next item, returns false when no more items
// are available or when an error occurs.
func (i *esIterator) Next() bool {
	if i.lastErr != nil || i.searchRes == nil ||
		i.cumIdx >= i.searchRes.Hits.Total.Count {

		return false
	}

	// Check if we need to fetch the next result batch and if so, update
	// the search request cursor and perform a new search.
	if i.searchResIdx >= len(i.searchRes.Hits.HitList) {
		i.searchReq["from"] = i.searchReq["from"].(uint64) + batchSize
		i.searchRes, i.lastErr = performSearch(i.client, i.searchReq)
		if i.lastErr != nil {
			return false
		}

		// Reset the search result index for the new page of results.
		i.searchResIdx = 0
	}

	i.doc = esDocToDoc(&i.searchRes.Hits.HitList[i.searchResIdx].DocSource)
	i.searchResIdx++
	i.cumIdx++

	return true
}

// Document returns the current document from the result set.
func (i *esIterator) Document() *index.Document {
	return i.doc
}

// TotalCount returns the approximated total number of search results.
func (i *esIterator) TotalCount() uint64 {
	if i.searchRes == nil {
		return 0
	}

	return i.searchRes.Hits.Total.Count
}

// Error returns the last error encountered by the iterator.
func (i *esIterator) Error() error {
	return i.lastErr
}

// Close releases any resources allocated to the iterator.
func (i *esIterator) Close() error {
	i.client = nil
	i.searchReq = nil
	i.cumIdx = i.searchRes.Hits.Total.Count

	return nil
}
