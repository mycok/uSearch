package crawler

import (
	"context"

	"github.com/mycok/uSearch/internal/graphlink/graph"
	"github.com/mycok/uSearch/internal/pipeline"
)

// Compile-time check for ensuring linkSource implements pipeline.Source.
var _ pipeline.Source = (*linkSource)(nil)

type linkSource struct {
	linkIt graph.LinkIterator
}

// Next returns a boolean to indicate the presence or absence of a next item.
func (ls *linkSource) Next(context.Context) bool {
	return ls.linkIt.Next()
}

// Error returns the last encountered error during the subsequent calls to Next.
func (ls *linkSource) Error() error {
	return ls.linkIt.Error()
}

// Payload returns the next payload after a successful call to Next
func (ls *linkSource) Payload() pipeline.Payload {
	// Note: we populate the payload with values from the retrieved link, all the
	// remaining payload fields are populated during the pipeline processing phase.
	link := ls.linkIt.Link()
	p := payloadPool.Get().(*crawlerPayload)

	p.LinkID = link.ID
	p.URL = link.URL
	p.RetrievedAt = link.RetrievedAt

	return p
}
