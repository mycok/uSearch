package es

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/mycok/uSearch/internal/textindexer/index"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/esapi"
)

// Compile-time check to ensure ElasticSearchIndexer implements Indexer.
// var _ index.Indexer = (*ElasticsearchIndexer)(nil)

// The name of the elasticsearch index to use.
const indexName = "textIndexer"

// Size of each page of results that is cached locally by the iterator.
const batchSize = 10

var esMappings = `
	{
		"mappings": {
			"properties": {
				"LinkID": {"type": "keyword"},
				"URL": {"type": "keyword"},
				"Content": {"type": "text"},
				"Title": {"type": "text"},
				"IndexedAt": {"type": "date"},
				"PageRank": {"type": "double"},
			}
		}
	}`

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
	DocSource esDoc `json:"_source"`
}

type esDoc struct {
	LinkID    string    `json:"LinkID"`
	URL       string    `json:"URL"`
	Title     string    `json:"Title"`
	Content   string    `json:"Content"`
	IndexedAt time.Time `json:"IndexedAt"`
	PageRank  float64   `json:"PageRank,omitempty"`
}

type esUpdateRes struct {
	Result string `json:"result"`
}

type esErrorRes struct {
	Error esError `json:"error"`
}

// esError satisfies the Error interface by implementing Error() string function.
type esError struct {
	Type   string `json:"type"`
	Reason string `json:"reason"`
}

func (e esError) Error() string {
	return fmt.Sprintf("%s: %s", e.Type, e.Reason)
}

// ElasticSearchIndexer uses an elastic search instance to catalogue and search documents.
type ElasticsearchIndexer struct {
	es         *elasticsearch.Client
	refreshOpt func(*esapi.UpdateRequest)
}

// NewElasticsearchIndexer creates and returns a text indexer that
// uses an elasticsearch instance to index and query documents.
func NewElasticsearchIndexer(esNodes []string, syncUpdates bool) (*ElasticsearchIndexer, error) {
	cfg := elasticsearch.Config{
		Addresses: esNodes,
	}

	es, err := elasticsearch.NewClient(cfg)
	if err != nil {
		return nil, err
	}

	if err := createIndex(es); err != nil {
		return nil, err
	}

	refreshOpt := es.Update.WithRefresh("false")

	if syncUpdates {
		refreshOpt = es.Update.WithRefresh("true")
	}

	return &ElasticsearchIndexer{
		es:         es,
		refreshOpt: refreshOpt,
	}, nil
}

// Index inserts a new document to the index or updates the index entry
// for and existing document.
func (i *ElasticsearchIndexer) Index(doc *index.Document) error {
	if doc.LinKID == uuid.Nil {
		return fmt.Errorf("index: %w", index.ErrMissingLinkID)
	}

	var (
		buf   bytes.Buffer
		esDoc = makeEsDoc(doc)
	)

	update := map[string]interface{}{
		"doc":           esDoc,
		"doc_as_upsert": true,
	}

	if err := json.NewEncoder(&buf).Encode(update); err != nil {
		return fmt.Errorf("index: %w", err)
	}

	res, err := i.es.Update(indexName, esDoc.LinkID, &buf, i.refreshOpt)
	if err != nil {
		return fmt.Errorf("index: %w", err)
	}

	var updateRes esUpdateRes
	if err = unmarshalResponse(res, &updateRes); err != nil {
		return fmt.Errorf("index: %w", err)
	}

	return nil
}

func createIndex(es *elasticsearch.Client) error {
	mappingsReader := strings.NewReader(esMappings)

	res, err := es.Indices.Create(indexName, es.Indices.Create.WithBody(mappingsReader))
	if err != nil {
		return fmt.Errorf("failed to create ES index: %w", err)
	} else if res.IsError() {
		err = unmarshalError(res)

		if esErr, valid := err.(esError); valid && esErr.Type == "resource_already_exists_exception" {
			return nil
		}

		return fmt.Errorf("failed to create ES index: %w", err)
	}

	return nil
}

func unmarshalError(res *esapi.Response) error {
	return unmarshalResponse(res, nil)
}

func unmarshalResponse(res *esapi.Response, to interface{}) error {
	defer func() {
		_ = res.Body.Close()
	}()

	if res.IsError() {
		var errRes esErrorRes

		if err := json.NewDecoder(res.Body).Decode(&errRes); err != nil {
			return err
		}

		return errRes.Error
	}

	return json.NewDecoder(res.Body).Decode(to)
}

func runSearch(es *elasticsearch.Client, searchQuery map[string]interface{}) (*esSearchRes, error) {
	return nil, nil
}

func mapEsDoc(d *esDoc) *index.Document {
	return &index.Document{
		LinKID:    uuid.MustParse(d.LinkID),
		URL:       d.URL,
		Title:     d.Title,
		Content:   d.Content,
		IndexedAt: d.IndexedAt.UTC(),
		PageRank:  d.PageRank,
	}
}

func makeEsDoc(d *index.Document) esDoc {
	// We intentionally skip PageRank as we don't want updates to
	// overwrite existing PageRank values.
	return esDoc{
		LinkID:    d.LinKID.String(),
		URL:       d.URL,
		Title:     d.Title,
		Content:   d.Content,
		IndexedAt: d.IndexedAt.UTC(),
	}
}