package crawler

import (
	"context"
	"time"

	"github.com/mycok/uSearch/internal/pipeline"
	"github.com/mycok/uSearch/internal/textindexer/index"
)

// Compile-time check for ensuring textIndexer implements pipeline.Processor.
var _ pipeline.Processor = (*textIndexer)(nil)

type textIndexer struct {
	indexer MiniIndexer
}

func newTextIndexer(indexer MiniIndexer) *textIndexer {
	return &textIndexer{indexer: indexer}
}

// Process creates a document instance from the passed payload and indexes the updated / new content
// by calling Index function.
func (ti *textIndexer) Process(ctx context.Context, p pipeline.Payload) (pipeline.Payload, error) {
	payload := p.(*crawlerPayload)

	doc := &index.Document{
		LinKID:    payload.LinkID,
		URL:       payload.URL,
		Title:     payload.Title,
		Content:   payload.TextContent,
		IndexedAt: time.Now(),
	}

	if err := ti.indexer.Index(doc); err != nil {
		return nil, err
	}

	return p, nil
}
