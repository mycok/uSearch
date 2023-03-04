package es

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/esapi"
	"github.com/google/uuid"

	"github.com/mycok/uSearch/textindexer/index"
)

// Static and compile-time check to ensure ElasticsearchIndex implements Indexer.
var _ index.Indexer = (*ElasticsearchIndex)(nil)

// Size of each page of results that is cached locally by the iterator.
const batchSize = 10

// The name of the elasticsearch index to use.
const indexName = "textindexer"

// JSON data structure that defines the properties of an elasticsearch
// document.
var esMappings = `
{
  "mappings" : {
    "properties": {
      "LinkID": {"type": "keyword"},
      "URL": {"type": "keyword"},
      "Content": {"type": "text"},
      "Title": {"type": "text"},
      "IndexedAt": {"type": "date"},
      "PageRank": {"type": "double"}
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
	PageRank  float64   `json:"PageRank,omitempty"`
	IndexedAt time.Time `json:"IndexedAt"`
}

type esUpdateRes struct {
	Result string `json:"result"`
}

type esErrorRes struct {
	Error esError `json:"error"`
}

type esError struct {
	Type   string `json:"type"`
	Reason string `json:"reason"`
}

func (e esError) Error() string {
	return fmt.Sprintf("%s: %s", e.Type, e.Reason)
}

// ElasticsearchIndex is an Indexer implementation that uses elasticsearch
// to index / catalogue and search documents.
type ElasticsearchIndex struct {
	client      *elasticsearch.Client
	refreshOpts func(*esapi.UpdateRequest)
}

// NewEsIndexer instantiates and returns an index that
// uses an elasticsearch instance to index and query documents.
func NewEsIndexer(
	esNodes []string, shouldSyncUpdates bool,
) (*ElasticsearchIndex, error) {

	cfg := elasticsearch.Config{
		Addresses: esNodes,
	}

	c, err := elasticsearch.NewClient(cfg)
	if err != nil {
		return nil, err
	}

	if err = initIndex(c); err != nil {
		return nil, err
	}

	refreshOpts := c.Update.WithRefresh("false")

	if shouldSyncUpdates {
		refreshOpts = c.Update.WithRefresh("true")
	}

	return &ElasticsearchIndex{
		client:      c,
		refreshOpts: refreshOpts,
	}, nil
}

// Index adds a new document or updates an existing index entry
// in case of an existing document.
func (s *ElasticsearchIndex) Index(doc *index.Document) error {
	if doc.LinkID == uuid.Nil {
		return fmt.Errorf("index: %w", index.ErrMissingLinkID)
	}

	var (
		buf   bytes.Buffer
		esDoc = makeEsDoc(doc)
	)

	forUpdate := map[string]interface{}{
		"doc":           esDoc,
		"doc_as_upsert": true,
	}

	if err := json.NewEncoder(&buf).Encode(forUpdate); err != nil {
		return fmt.Errorf("index: %w", err)
	}

	res, err := s.client.Update(indexName, esDoc.LinkID, &buf, s.refreshOpts)
	if err != nil {
		return fmt.Errorf("index: %w", err)
	}

	var updateRes esUpdateRes
	if err = json.NewDecoder(res.Body).Decode(&updateRes); err != nil {
		return fmt.Errorf("index: %w", err)
	}

	return nil
}

// FindByID looks up a document by its link ID.
func (s *ElasticsearchIndex) FindByID(linkID uuid.UUID) (*index.Document, error) {
	query := map[string]interface{}{
		"query": map[string]interface{}{
			"match": map[string]interface{}{
				"LinkID": linkID.String(),
			},
		},
		"from": 0,
		"size": 1,
	}

	searchRes, err := performSearch(s.client, query)
	if err != nil {
		return nil, fmt.Errorf("find by ID: %w", err)
	}

	if len(searchRes.Hits.HitList) == 0 {
		return nil, fmt.Errorf("find by ID: %w", index.ErrNotFound)
	}

	return esDocToDoc(&searchRes.Hits.HitList[0].DocSource), nil
}

// Search performs a look up based on query and returns a result
// iterator if successful or an error otherwise.
func (s *ElasticsearchIndex) Search(q index.Query) (index.Iterator, error) {
	var QueryType string

	switch q.Type {
	case index.QueryTypePhrase:
		QueryType = "phrase"
	default:
		QueryType = "best_fields"
	}

	query := map[string]interface{}{
		"query": map[string]interface{}{
			"function_score": map[string]interface{}{
				"query": map[string]interface{}{
					"multi_match": map[string]interface{}{
						"type":   QueryType,
						"query":  q.Expression,
						"fields": []string{"Title", "Content"},
					},
				},
				"script_score": map[string]interface{}{
					"script": map[string]interface{}{
						"source": "_score + doc['PageRank'].value",
					},
				},
			},
		},
		"from": q.Offset,
		"size": batchSize,
	}

	searchRes, err := performSearch(s.client, query)
	if err != nil {
		return nil, fmt.Errorf("search: %w", err)
	}

	return &esIterator{
		client:    s.client,
		searchReq: query,
		searchRes: searchRes,
		cumIdx:    q.Offset,
	}, nil
}

// UpdateScore updates the PageRank score for a document with the
// specified link ID. If no such document exists, a placeholder
// document with the provided score will be created.
func (s *ElasticsearchIndex) UpdateScore(linkID uuid.UUID, score float64) error {
	var buf bytes.Buffer

	updateQuery := map[string]interface{}{
		"doc": map[string]interface{}{
			"LinkID":   linkID.String(),
			"PageRank": score,
		},
		"doc_as_upsert": true,
	}

	if err := json.NewEncoder(&buf).Encode(updateQuery); err != nil {
		return fmt.Errorf("update score: %w", err)
	}

	res, err := s.client.Update(indexName, linkID.String(), &buf, s.refreshOpts)
	if err != nil {
		return fmt.Errorf("update score: %w", err)
	}

	var updateRes esUpdateRes
	if err = unmarshalResponse(res, &updateRes); err != nil {
		return fmt.Errorf("update score: %w", err)
	}

	return nil
}

func performSearch(
	client *elasticsearch.Client, query map[string]interface{},
) (*esSearchRes, error) {
	var buf bytes.Buffer

	if err := json.NewEncoder(&buf).Encode(query); err != nil {
		return nil, fmt.Errorf("find by id: %w", err)
	}

	// Perform the search.
	res, err := client.Search(
		client.Search.WithContext(context.Background()),
		client.Search.WithIndex(indexName),
		client.Search.WithBody(&buf),
	)
	if err != nil {
		return nil, err
	}

	var esRes esSearchRes
	if err = unmarshalResponse(res, &esRes); err != nil {
		return nil, err
	}

	return &esRes, nil
}

func initIndex(client *elasticsearch.Client) error {
	mappingsReader := strings.NewReader(esMappings)

	res, err := client.Indices.Create(
		indexName,
		client.Indices.Create.WithBody(mappingsReader),
	)
	// For cases where index creation fails due to client issues,
	// ie network connection issues
	if err != nil {
		return fmt.Errorf("failed to create ES index: %w", err)
	}

	// For cases where index creation fails due to other issues, ie invalid params.
	if res.IsError() {
		err = unMarshalError(res)

		esErr, isSuccessful := err.(esError)
		if isSuccessful && esErr.Type == "resource_already_exists_exception" {
			return nil
		}

		return fmt.Errorf("failed to create ES index: %w", err)
	}

	return nil
}

func unMarshalError(res *esapi.Response) error {
	return unmarshalResponse(res, nil)
}

func unmarshalResponse(res *esapi.Response, into interface{}) error {
	defer func() {
		res.Body.Close()
	}()

	if res.IsError() {
		var errRes esErrorRes
		if err := json.NewDecoder(res.Body).Decode(&errRes); err != nil {
			return err
		}

		return errRes.Error
	}

	return json.NewDecoder(res.Body).Decode(into)
}

func esDocToDoc(doc *esDoc) *index.Document {
	return &index.Document{
		LinkID:    uuid.MustParse(doc.LinkID),
		URL:       doc.URL,
		Title:     doc.Title,
		Content:   doc.Content,
		PageRank:  doc.PageRank,
		IndexedAt: doc.IndexedAt.UTC(),
	}
}

func makeEsDoc(doc *index.Document) esDoc {
	// We intentionally skip PageRank as we don't want updates to
	// overwrite existing PageRank values.
	return esDoc{
		LinkID:    doc.LinkID.String(),
		URL:       doc.URL,
		Title:     doc.Title,
		Content:   doc.Content,
		IndexedAt: doc.IndexedAt.UTC(),
	}
}
