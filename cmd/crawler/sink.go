package crawler

import (
	"context"

	"github.com/mycok/uSearch/internal/pipeline"
)

// Compile-time check for ensuring sink implements pipeline.Sink.
var _ pipeline.Sink = (*sink)(nil)

type sink struct {
	count int
}

// Consume ignores the provided payload and returns nil. The [pipeline.Process] method
// calls [Payload.MarkAsProcessed] method which reset the payload values and adds it to the payloadPool
// for future re-use.
func (s *sink) Consume(context.Context, pipeline.Payload) error {
	s.count++

	return nil
}

func (s *sink) getCount() int {
	return s.count / 2
}
