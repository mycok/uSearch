package es

import (
	"github.com/mycok/uSearch/internal/textindexer/index"

	"github.com/elastic/go-elasticsearch/v8"
)

// esIterator implements index.Iterator
type esIterator struct {
	es        *elasticsearch.Client
	searchReq map[string]interface{}
	cumIdx    uint64
	srIdx     int
	sr        *esSearchRes
	doc       *index.Document
	lastErr   error
}

// Close implements index.Iterator.Close() and releases any allocated resources
// still being used by the iterator.
func (i *esIterator) Close() error {
	i.es = nil
	i.searchReq = nil
	i.cumIdx = i.sr.Hits.Total.Count

	return nil
}

// Next implements index.Iterator.Next(). loads the next document matching the search query.
// It returns false if no more documents are available.
func (i *esIterator) Next() bool {
	if i.lastErr != nil || i.sr == nil || i.cumIdx >= i.sr.Hits.Total.Count {
		return false
	}

	// Check if we need to fetch the next result batch.
	if i.srIdx >= len(i.sr.Hits.HitList) {
		i.searchReq["from"] = i.searchReq["from"].(uint64) + batchSize
		if i.sr, i.lastErr = runSearch(i.es, i.searchReq); i.lastErr != nil {
			return false
		}

		i.srIdx = 0
	}

	i.doc = mapEsDoc(&i.sr.Hits.HitList[i.srIdx].DocSource)
	i.cumIdx++
	i.srIdx++

	return true
}

// Error implements index.Iterator.Error() and returns the last error
// returned by calling index.Iterator.Next().
func (i *esIterator) Error() error {
	return i.lastErr
}

// Document implements index.Iterator.Document() and returns the
// current document from the search result set.
func (i *esIterator) Document() *index.Document {
	return i.doc
}

// TotalCount implements index.Iterator.TotalCount() and returns
// the approximate number of search results.
func (i *esIterator) TotalCount() uint64 {
	return i.sr.Hits.Total.Count
}
