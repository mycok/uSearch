package crawler

import (
	"bytes"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/mycok/uSearch/pipeline"
)

var (
	_ pipeline.Payload = (*crawlerPayload)(nil)

	payloadPool = sync.Pool{
		New: func() interface{} {
			return new(crawlerPayload)
		},
	}
)

type crawlerPayload struct {
	LinkID        uuid.UUID    // populated by the input source (pipeline.Source) type.
	URL           string       // populated by the input source (pipeline.Source) type.
	RetrievedAt   time.Time    // populated by the input source (pipeline.Source) type.
	RawContent    bytes.Buffer // populated by the link fetcher type.
	NoFollowLinks []string     // populated by the link extractor type.
	Links         []string     // populated by the link extractor type.
	Title         string       // populated by the text extractor type.
	TextContent   string       // populated by the text extractor type.
}

// Clone returns a deep-copy of the original payload.
func (p *crawlerPayload) Clone() pipeline.Payload {
	payloadClone := payloadPool.Get().(*crawlerPayload)

	payloadClone.LinkID = p.LinkID
	payloadClone.URL = p.URL
	payloadClone.RetrievedAt = p.RetrievedAt
	payloadClone.NoFollowLinks = append([]string(nil), payloadClone.NoFollowLinks...)
	payloadClone.Links = append([]string(nil), payloadClone.Links...)
	payloadClone.Title = p.Title
	payloadClone.TextContent = p.TextContent

	_, err := io.Copy(&payloadClone.RawContent, &p.RawContent)
	if err != nil {
		panic(fmt.Sprintf("[BUG]::error cloning payload raw content: %v", err))
	}

	return payloadClone
}

// MarkAsProcessed is invoked by the stage / stage runner when the payload
// either reaches the pipeline sink or it gets discarded by one of the
// pipeline stages.
func (p *crawlerPayload) MarkAsProcessed() {
	p.URL = p.URL[:0]
	p.RawContent.Reset()
	p.NoFollowLinks = p.NoFollowLinks[:0]
	p.Links = p.Links[:0]
	p.Title = p.Title[:0]
	p.TextContent = p.TextContent[:0]

	// Put back a reset pointer to crawler payload into the pool for re-use.
	payloadPool.Put(p)
}
