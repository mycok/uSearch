package pipeline_test

import (
	"context"
	"fmt"
	// "sort"
	// "time"

	"github.com/mycok/uSearch/internal/pipeline"

	check "gopkg.in/check.v1"
)

// Initialize and register an instance of StageTestSuite to be
// executed by check testing package.
var _ = check.Suite(new(StageTestSuite))

type StageTestSuite struct {}

func (s *StageTestSuite) TestFIFO(c *check.C) {
	stages := make([]pipeline.StageRunner, 10)
	for i := 0; i < len(stages); i++ {
		stages[i] = pipeline.FIFO(makePassThruProcessor())
	}

	src := &sourceStab{data: stringPayloads(3)}
	sink := new(sinkStub)

	p := pipeline.New(stages...)
	err := p.Process(context.TODO(), src, sink)

	c.Assert(err, check.IsNil)
	c.Assert(sink.data, check.DeepEquals, src.data)
	assertAllProcessed(c, src.data)
}

// Test suite setup helpers.
func makeMutatingProcessor(index int) pipeline.Processor {
	return pipeline.ProcessorFunc(func(_ context.Context, p pipeline.Payload) (pipeline.Payload, error) {
		// Mutate payload to check that each processor got a copy.
		sp := p.(*stringPayload)
		sp.val = fmt.Sprintf("%s_%d", sp.val, index)

		return p, nil
	})
}

func makePassThruProcessor() pipeline.Processor {
	return pipeline.ProcessorFunc(func(_ context.Context, p pipeline.Payload) (pipeline.Payload, error) {
		return p, nil
	})
}