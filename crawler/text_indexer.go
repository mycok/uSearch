package crawler

import (
	"context"
	"time"

	"github.com/mycok/uSearch/pipeline"
	"github.com/mycok/uSearch/textindexer/index"
)

// Static and compile-time check to ensure textIndexer implements
// pipeline.Processor interface.
var _ pipeline.Processor = (*textIndexer)(nil)

type textIndexer struct {
	indexer MiniIndexer
}

func newTextIndexer(indexer MiniIndexer) *textIndexer {
	return &textIndexer{indexer}
}

// Process inserts / updates the index using the payloads new / updated values.
func (p *textIndexer) Process(
	ctx context.Context, payload pipeline.Payload,
) (pipeline.Payload, error) {

	cPayload, ok := payload.(*crawlerPayload)
	if !ok {
		return nil, nil
	}

	doc := &index.Document{
		LinkID:    cPayload.LinkID,
		URL:       cPayload.URL,
		Title:     cPayload.Title,
		Content:   cPayload.TextContent,
		IndexedAt: time.Now(),
	}

	if err := p.indexer.Index(doc); err != nil {
		return nil, err
	}

	return cPayload, nil
}
