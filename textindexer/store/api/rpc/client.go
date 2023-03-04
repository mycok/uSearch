package rpc

import (
	"context"
	"errors"
	"io"

	"github.com/google/uuid"

	"github.com/mycok/uSearch/textindexer/index"
	"github.com/mycok/uSearch/textindexer/store/api/rpc/proto"
)

// TextIndexerClient provides an API that wraps the index.Indexer interface
// for accessing index data store instances exposed by a remote gRPC server.
type TextIndexerClient struct {
	ctx       context.Context
	rpcClient proto.TextIndexerClient
}

// NewTextIndexerClient configures and returns a TextIndexerClient instance.
func NewTextIndexerClient(
	ctx context.Context, rpcClient proto.TextIndexerClient,
) *TextIndexerClient {

	return &TextIndexerClient{
		ctx:       ctx,
		rpcClient: rpcClient,
	}
}


// Index adds a new document or updates an existing index entry
// in case of an existing document.
func (c *TextIndexerClient) Index(doc *index.Document) error {
	req := &proto.Document{
		LinkId:  doc.LinkID[:],
		Url:     doc.URL,
		Title:   doc.Title,
		Content: doc.Content,
	}

	result, err := c.rpcClient.Index(c.ctx, req)
	if err != nil {
		return err
	}

	doc.IndexedAt = result.IndexedAt.AsTime()

	return nil
}

// Search performs a look up based on query and returns a result
// iterator if successful or an error otherwise.
func (c *TextIndexerClient)	Search(q index.Query) (index.Iterator, error) {
	req := &proto.Query{
		Type: proto.Query_Type(q.Type),
		Expression: q.Expression,
		Offset: q.Offset,
	}

	ctx, cancel := context.WithCancel(c.ctx)
	
	stream, err := c.rpcClient.Search(ctx, req)
	if err != nil {
		// Cancel the context
		cancel()

		return nil, err
	}

	result, err := stream.Recv()
	if err != nil {
		// Cancel the context
		cancel()

		return nil, err
	} else if result.GetDoc() != nil {
		// Cancel the context
		cancel()

		return nil, errors.New(
			"expected result count value before sending any documents",
		)
	}

	return &docIterator{
		total: result.GetDocCount(),
		stream: stream,
		cancelFn: cancel,
	}, nil
}

// UpdateScore updates the PageRank score for a document with the
// specified link ID. If no such document exists, a placeholder
// document with the provided score will be created.
func (c *TextIndexerClient)	UpdateScore(linkID uuid.UUID, score float64) error {
	req := &proto.UpdateScoreRequest{
		LinkId: linkID[:],
		PageRankScore: score,
	}

	_, err := c.rpcClient.UpdateScore(c.ctx, req)

	return err
}

type docIterator struct {
	total uint64
	stream   proto.TextIndexer_SearchClient
	doc     *index.Document
	lastErr  error
	cancelFn func()
}

// Next loads the next item, returns false when no more docs
// are available or when an error occurs.
func (i *docIterator) Next() bool {
	result, err := i.stream.Recv()
	if err != nil {
		if err == io.EOF {
			i.lastErr = err
		}

		// Cancel the provided context.
		i.cancelFn()

		return false
	}

	doc := result.GetDoc()
	if doc == nil {
		// Cancel the provided context.
		i.cancelFn()

		i.lastErr = errors.New("received no document from search result")

		return false
	}

	i.doc = &index.Document{
		LinkID:          uuidFromBytes(doc.LinkId),
		URL:         doc.Url,
		Title: doc.Title,
		Content: doc.Content,
		IndexedAt: doc.IndexedAt.AsTime(),
	}

	return true
}

// Error returns the last error encountered by the iterator.
func (i *docIterator) Error() error {
	return i.lastErr
}

// Close releases any resources allocated to the iterator.
func (i *docIterator) Close() error {
	// Cancel the context.
	i.cancelFn()

	return nil
}

// Link returns the currently fetched link object.
func (i *docIterator) Document() *index.Document {
	return i.doc
}

// TotalCount returns the approximated total number of search results.
func (i *docIterator) TotalCount() uint64 {
	return i.total
}
