package pipeline_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/mycok/uSearch/internal/pipeline"

	check "gopkg.in/check.v1"
)

// Initialize and register an instance of the PipelineTestSuite to be
// executed by check testing package.
var _ = check.Suite(new(PipelineTestSuite))

type PipelineTestSuite struct {}

func Test(t *testing.T) {
	check.TestingT(t)
}

func (s *PipelineTestSuite) TestDataFlow(c *check.C) {
	stages := make([]pipeline.StageRunner, 10)
	for i := 0; i < len(stages); i++ {
		stages[i] = testStage{c: c}
	}

	src := &sourceStab{data: stringPayloads(3)}
	sink := new(sinkStub)

	p := pipeline.New(stages...)
	err := p.Process(context.TODO(), src, sink)

	c.Assert(err, check.IsNil)
	c.Assert(sink.data, check.DeepEquals, src.data)
	assertAllProcessed(c, src.data)
}

func (s *PipelineTestSuite) TestProcessorErrHandling(c *check.C) {
	processErr := errors.New("some error")
	stages := make([]pipeline.StageRunner, 10)

	for i := 0; i < len(stages); i++ {
		var err error
		if i == 5 {
			err = processErr
		}
		// Pass a stage runner with an error and check that it's returned by the pipeline process
		// method by reading from the error buffered channel. 
		stages[i] = testStage{c: c, err: err}
	}

	src := &sourceStab{data: stringPayloads(3)}
	sink := new(sinkStub)

	p := pipeline.New(stages...)
	err := p.Process(context.TODO(), src, sink)

	c.Assert(err, check.ErrorMatches, "(?s).*some error.*")
}

func (s *PipelineTestSuite) TestSourceErrHandling(c *check.C) {
	processErr := errors.New("some error")
	// Pass a source with an error and check that it's returned by the pipeline process
	// method by reading from the error buffered channel. 
	src := &sourceStab{
		data: stringPayloads(3),
		err: processErr,
	}

	sink := new(sinkStub)

	p := pipeline.New(testStage{c: c})
	err := p.Process(context.TODO(), src, sink)

	c.Assert(err, check.ErrorMatches, "(?s).*pipeline source: some error.*")
}

// Test setup helpers
func assertAllProcessed(c *check.C, payloads []pipeline.Payload) {
	for i, p := range payloads {
		payload := p.(*stringPayload)
		c.Assert(payload.processed, check.Equals, true, check.Commentf("payload %d not processed", i))
	}
}

type testStage struct {
	c *check.C
	dropPayloads bool
	err error
}

func (s testStage) Run(ctx context.Context, params pipeline.StageParams) {
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

			if s.dropPayloads {
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

// .......
type sourceStab struct {
	index int
	data []pipeline.Payload
	err error
}

func (s *sourceStab) Next(ctx context.Context) bool {
	if s.err != nil || s.index == len(s.data) {
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

// .....
type sinkStub struct {
	data []pipeline.Payload
	err error
}

func (s *sinkStub) Consume(ctx context.Context, p pipeline.Payload) error {
	s.data = append(s.data, p)

	return s.err
}

// .......
type stringPayload struct {
	processed bool
	val string
}

func (s *stringPayload) Clone() pipeline.Payload {
	return &stringPayload{
		val: s.val,
	}
}

func (s *stringPayload) MarkAsProcessed() {
	s.processed = true
}

func (s *stringPayload) String() string {
	return s.val
}

func stringPayloads(numOfValues int) []pipeline.Payload {
	out := make([]pipeline.Payload, numOfValues)
	for i := 0; i < len(out); i++ {
		out[i] = &stringPayload{
			val: fmt.Sprint(i),
		}
	}

	return out
}
