package crawler

import (
	"context"

	"github.com/mycok/uSearch/linkgraph/graph"
	"github.com/mycok/uSearch/pipeline"
)

// Static and compile-time check to ensure linkSource implements
// pipeline.Source interface.
var _ pipeline.Source = (*linkSource)(nil)

type linkSource struct {
	linkIt graph.LinkIterator
}

// Next loads the next available payload from the source and returns true.
// When no more payloads are available or an error occurs, calls to Next
// return false.
func (s *linkSource) Next(ctx context.Context) bool {
	return s.linkIt.Next()
}

// Payload returns the current payload to be processed.
func (s *linkSource) Payload() pipeline.Payload {
	payload := payloadPool.Get().(*crawlerPayload)
	link := s.linkIt.Link()

	// Note: we populate the payload with some values from the retrieved link,
	// all the remaining payload fields are populated by the various pipeline
	// stages during pipeline execution.
	payload.LinkID = link.ID
	payload.URL = link.URL
	payload.RetrievedAt = link.RetrievedAt

	return payload
}

// Error returns the last error encountered by the source.
func (s *linkSource) Error() error {
	return s.linkIt.Error()
}
