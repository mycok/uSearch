package crawler

import (
	"bytes"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/mycok/uSearch/internal/pipeline"

	"github.com/google/uuid"
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
	LinkID        uuid.UUID
	URL           string
	RetrievedAt   time.Time
	RawContent    bytes.Buffer
	NoFollowLinks []string // NoFollowLinks are still added to the graph but no outgoing edges will be created from this link to them.
	Links         []string
	Title         string
	TextContent   string
}

// Clone implements pipeline.Payload. it returns a copy of the crawlerPayload.
func (p *crawlerPayload) Clone() pipeline.Payload {
	newP := payloadPool.Get().(*crawlerPayload)
	newP.LinkID = p.LinkID
	newP.URL = p.URL
	newP.RetrievedAt = p.RetrievedAt
	newP.NoFollowLinks = append([]string(nil), p.NoFollowLinks...)
	newP.Links = append([]string(nil), p.Links...)
	newP.Title = p.Title
	newP.TextContent = p.TextContent

	_, err := io.Copy(&newP.RawContent, &p.RawContent)
	if err != nil {
		panic(fmt.Sprintf("[BUG]: error cloning payload raw content: %v", err))
	}

	return newP
}

// MarkAsProcessed implements pipeline.payload. it resets the crawlerPayload fields and put the
// crawlerPayload instance into the payloadPool for future re-use.
func (p *crawlerPayload) MarkAsProcessed() {
	p.URL = p.URL[:0]
	p.RawContent.Reset()
	p.NoFollowLinks = p.NoFollowLinks[:0]
	p.Links = p.Links[:0]
	p.Title = p.Title[:0]
	p.TextContent = p.TextContent[:0]

	payloadPool.Put(p)
}
