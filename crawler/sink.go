package crawler

import (
	"context"

	"github.com/mycok/uSearch/pipeline"
)

// Static and compile-time check to ensure countingSink implements
// pipeline.Sink interface.
var _ pipeline.Sink = (*countingSink)(nil)

type countingSink struct {
	count int
}

func (s *countingSink) Consume(ctx context.Context, p pipeline.Payload) error {
	s.count++

	return nil
}

func (s *countingSink) getCount() int {
	// The broadcast split-stage sends out two payloads for each incoming link
	// so we need to divide the total count by 2.
	return s.count / 2
}
