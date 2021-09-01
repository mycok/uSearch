package es

import (
	"time"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/mycok/uSearch/internal/textindexer/index"
)

// The name of the elasticsearch index to use.
const indexName = "textIndexer"

// Size of each page of results that is cached locally by the iterator.
const batchSize = 10

type esSearchRes struct {
	Hits esSearchResHits `json:"hits"`
}

type esSearchResHits struct {
	Total   esTotal        `json:"total"`
	HitList []esHitWrapper `json:"hits"`
}

type esTotal struct {
	Count uint64 `json:"value"`
}

type esHitWrapper struct {
	DocSource esDoc `json:"_source`
}

type esDoc struct {
	LinkID    string    `json:"LinkID"`
	URL       string    `json:"URL"`
	Title     string    `json:"Title"`
	Content   string    `json:"Content"`
	IndexedAt time.Time `json:"IndexedAt"`
	PageRank  float64   `json:"PageRank,omitempty"`
}

func runSearch(es *elasticsearch.Client, searchQuery map[string]interface{}) (*esSearchRes, error) {
	return nil, nil
}

func mapEsDoc(doc *esDoc) *index.Document {
	return nil
}
