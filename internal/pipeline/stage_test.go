package pipeline_test

import (
	"context"
	"fmt"
	"time"

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

func (s *StageTestSuite) TestFixedWorkerPool(c *check.C) {
	numOfWorkers := 5
	syncCh := make(chan struct{})
	rendevousCh := make(chan struct{})
	doneCh := make(chan struct{})

	proc := pipeline.ProcessorFunc(func(c context.Context, p pipeline.Payload) (pipeline.Payload, error) {
		// Signal that we have reached the sync point and wait for the
		// green light to proceed by the test code.
		syncCh <- struct{}{}
		<-rendevousCh

		return nil, nil
	})

	src := &sourceStab{data: stringPayloads(numOfWorkers)}
	p := pipeline.New(pipeline.FixedWorkerPool(proc, numOfWorkers))

	go func() {
		err := p.Process(context.TODO(), src, nil)

		c.Assert(err, check.IsNil)
		// Close the done channel after p.Process returns but before the worker returns / completes.
		// Closing the done channels signals it's enclosing worker / go-routine that it's time to exit.
		close(doneCh)
	}()

	// Wait for all workers to reach sync point. This means that each input
	// from the source is currently handled by a worker in parallel.
	for i := 0; i < numOfWorkers; i++ {
		select {
		case <-syncCh:
		case <-time.After(10 * time.Second):
			c.Fatalf("timed out waiting for worker %d to reach sync point", i)
		}
	}

	// Allow workers / go-routines to proceed and wait for the pipeline to complete.
	close(rendevousCh)
	select {
	case <-doneCh:
	case <-time.After(10 * time.Second):
		c.Fatal("timed out waiting for pipeline to complete")
	}
}

func (s *StageTestSuite) TestDynamicWorkerPool(c *check.C) {
	numOfWorkers := 5
	syncCh := make(chan struct{}, numOfWorkers)
	rendevousCh := make(chan struct{})
	doneCh := make(chan struct{})

	proc := pipeline.ProcessorFunc(func(c context.Context, p pipeline.Payload) (pipeline.Payload, error) {
		// Signal that we have reached the sync point and wait for the
		// green light to proceed by the test code.
		syncCh <- struct{}{}
		<-rendevousCh

		return nil, nil
	})

	src := &sourceStab{data: stringPayloads(numOfWorkers * 2)}
	p := pipeline.New(pipeline.DynamicWorkerPool(proc, numOfWorkers))

	go func() {
		err := p.Process(context.TODO(), src, nil)

		c.Assert(err, check.IsNil)
		// Close the done channel after p.Process returns but before the worker returns / completes.
		// Closing the done channels signals it's enclosing worker / go-routine that it's time to exit.
		close(doneCh)
	}()

	// Wait for all workers to reach sync point. This means that the pool
	// has scaled up to the max number of workers.
	for i := 0; i < numOfWorkers; i++ {
		select {
		case <-syncCh:
		case <-time.After(10 * time.Second):
			c.Fatalf("timed out waiting for worker %d to reach sync point", i)
		}
	}

	// Allow workers / go-routines to proceed and process the next batch of records.
	close(rendevousCh)
	select {
	case <-doneCh:
	case <-time.After(10 * time.Second):
		c.Fatalf("timed out waiting for pipeline to complete")
	}

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