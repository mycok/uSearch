package pipeline_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	check "gopkg.in/check.v1"

	"github.com/mycok/uSearch/pipeline"
)

// Initialize and register a pointer instance of the pipelineTestSuite to be
// executed by check testing package.
var _ = check.Suite(new(pipelineTestSuite))

// Test registers the [check] library with the go testing library and enables
// the running of the test suite using the go testing library.
func Test(t *testing.T) {
	check.TestingT(t)
}

type pipelineTestSuite struct{}

func (s *pipelineTestSuite) TestDataflow(c *check.C) {
	stages := make([]pipeline.StageRunner, 10)
	for i := 0; i < len(stages); i++ {
		stages[i] = &stageRunnerStab{c: c}
	}

	src := &sourceStab{data: generateStringPayloads(3)}
	sink := new(sinkStab)
	p := pipeline.New(stages...)

	err := p.Execute(context.TODO(), src, sink)
	c.Assert(err, check.IsNil)
	c.Assert(src.data, check.DeepEquals, sink.data)
	assertAllPayloadProcessed(c, sink.data...)
}

func (s *pipelineTestSuite) TestStageRunnerProcessorErrHandling(c *check.C) {
	processorErr := errors.New("processor error")
	stages := make([]pipeline.StageRunner, 10)
	for i := 0; i < len(stages); i++ {
		var err error
		if i%5 == 0 {
			err = processorErr
		}

		stages[i] = &stageRunnerStab{
			c:   c,
			err: err,
		}
	}

	src := &sourceStab{data: generateStringPayloads(3)}
	sink := new(sinkStab)
	p := pipeline.New(stages...)

	err := p.Execute(context.TODO(), src, sink)
	c.Assert(err, check.ErrorMatches, "(?s).*processor error.*")
}

func (s *pipelineTestSuite) TestStageRunnerProcessorPayloadDrop(c *check.C) {
	src := &sourceStab{data: generateStringPayloads(1)}
	sink := new(sinkStab)
	p := pipeline.New(&stageRunnerStab{
		c:                  c,
		shouldDropPayloads: true,
	})

	err := p.Execute(context.TODO(), src, sink)
	c.Assert(err, check.IsNil)
	c.Assert(sink.data, check.HasLen, 0)
	assertAllPayloadProcessed(c, src.data...)
}

func (s *pipelineTestSuite) TestSourceErrHandling(c *check.C) {
	srcErr := errors.New("source error")
	// Instantiate a source with an error value and check that that error is
	// it's returned by the pipeline.Execute method after the pipeline execution
	// by reading from the error channel.
	src := &sourceStab{
		data: generateStringPayloads(3),
		err:  srcErr,
	}
	sink := new(sinkStab)
	p := pipeline.New(&stageRunnerStab{c: c})

	err := p.Execute(context.TODO(), src, sink)
	c.Assert(err, check.ErrorMatches, "(?s).*source error.*")
}

func (s *pipelineTestSuite) TestSinkErrHanding(c *check.C) {
	sinkErr := errors.New("sink error")
	src := &sourceStab{data: generateStringPayloads(1)}
	sink := &sinkStab{err: sinkErr}
	p := pipeline.New(&stageRunnerStab{c: c})

	err := p.Execute(context.TODO(), src, sink)
	c.Assert(err, check.ErrorMatches, "(?s).*sink error.*")
}

// Helper functions and stabs.
func assertAllPayloadProcessed(c *check.C, payloads ...pipeline.Payload) {
	for i, p := range payloads {
		payload := p.(*stringPayload)
		c.Assert(
			payload.isProcessed, check.Equals, true,
			check.Commentf("payload %d not processed", i),
		)
	}
}

type sourceStab struct {
	index int
	data  []pipeline.Payload
	err   error
}

func (s *sourceStab) Next(ctx context.Context) bool {
	if s.index >= len(s.data) || s.err != nil {
		return false
	}

	s.index++

	return true
}

func (s *sourceStab) Payload() pipeline.Payload {
	return s.data[s.index-1]
}

func (s *sourceStab) Error() error {
	return s.err
}

type sinkStab struct {
	data []pipeline.Payload
	err  error
}

func (s *sinkStab) Consume(ctx context.Context, p pipeline.Payload) error {
	s.data = append(s.data, p)

	return s.err
}

type stringPayload struct {
	value       string
	isProcessed bool
}

func (p *stringPayload) Clone() pipeline.Payload {
	return &stringPayload{value: p.value}
}

func (p *stringPayload) MarkAsProcessed() {
	p.isProcessed = true
}

func generateStringPayloads(numOfPayloads int) []pipeline.Payload {
	payloads := make([]pipeline.Payload, numOfPayloads)
	for i := 0; i < numOfPayloads; i++ {
		payloads[i] = &stringPayload{value: fmt.Sprint(i)}
	}

	return payloads
}

type stageRunnerStab struct {
	c                  *check.C
	shouldDropPayloads bool
	err                error
}

func (s *stageRunnerStab) Run(ctx context.Context, params pipeline.StageParams) {
	defer func() {
		s.c.Logf("[stage %d] exiting", params.StageIndex())
	}()

	for {
		select {
		case <-ctx.Done():
			return
		case payload, ok := <-params.Input():
			if !ok {
				return
			}

			s.c.Logf("[stage %d] received payload: %v", params.StageIndex(), payload)
			if s.err != nil {
				s.c.Logf("[stage %d] emitted error: %v", params.StageIndex(), s.err)
				params.Error() <- s.err

				return
			}

			if s.shouldDropPayloads {
				s.c.Logf("[stage %d] dropping payload: %v", params.StageIndex(), payload)
				payload.MarkAsProcessed()

				continue
			}

			s.c.Logf("[stage %d] emitting payload: %v", params.StageIndex(), payload)
			select {
			case <-ctx.Done():
				return
			case params.Output() <- payload:
			}
		}
	}
}
