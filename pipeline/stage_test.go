package pipeline_test

import (
	"context"
	"fmt"
	"sort"
	"time"

	check "gopkg.in/check.v1"

	"github.com/mycok/uSearch/pipeline"
)

// Initialize and register a pointer instance of the StageTestSuite to be
// executed by check testing package.
var _ = check.Suite(new(stageRunnerTestSuite))

type stageRunnerTestSuite struct{}

func (s *stageRunnerTestSuite) TestFIFO(c *check.C) {
	stages := make([]pipeline.StageRunner, 10)
	for i := 0; i < len(stages); i++ {
		stages[i] = pipeline.NewFIFO(generatePassThruProcessor())
	}

	src := &sourceStab{data: generateStringPayloads(3)}
	sink := new(sinkStab)
	p := pipeline.New(stages...)

	err := p.Execute(context.TODO(), src, sink)
	c.Assert(err, check.IsNil)
	c.Assert(src.data, check.DeepEquals, sink.data)
	assertAllPayloadProcessed(c, src.data...)
}

func (s *stageRunnerTestSuite) TestFixedWorkerPool(c *check.C) {
	numOfWorkers := 10
	syncChan := make(chan struct{})
	rendezvousChan := make(chan struct{})
	doneChan := make(chan struct{})
	// This processor function discards the payload.
	proc := pipeline.ProcessorFunc(
		func(ctx context.Context, p pipeline.Payload) (pipeline.Payload, error) {
			syncChan <- struct{}{}
			<-rendezvousChan

			return nil, nil
		})

	src := &sourceStab{data: generateStringPayloads(numOfWorkers)}
	p := pipeline.New(pipeline.NewFixedWorkerPool(proc, numOfWorkers))

	go func() {
		err := p.Execute(context.TODO(), src, nil)
		c.Assert(err, check.IsNil)

		close(doneChan)
	}()

	// Wait for all workers to reach sync point. This means that each input
	// from the source is currently handled by a worker in parallel.
	for i := 0; i < numOfWorkers; i++ {
		select {
		case <-syncChan:
		case <-time.After(10 * time.Second):
			c.Fatalf("timed out waiting for worker %d to reach sync point", i)
		}
	}

	// Allow workers / go-routines to proceed.
	close(rendezvousChan)

	// Wait for the pipeline to complete.
	select {
	case <-doneChan:
		close(syncChan)
	case <-time.After(10 * time.Second):
		c.Fatal("timed out waiting for pipeline to complete")
	}
}

func (s *stageRunnerTestSuite) TestDynamicWorkerPool(c *check.C) {
	var numOfExecutedProcesses int
	maxNumOfWorkers := 5
	syncChan := make(chan struct{}, maxNumOfWorkers)
	rendezvousChan := make(chan struct{})
	doneChan := make(chan struct{})
	// This processor function discards the payload.
	proc := pipeline.ProcessorFunc(
		func(ctx context.Context, p pipeline.Payload) (pipeline.Payload, error) {
			syncChan <- struct{}{}
			<-rendezvousChan

			numOfExecutedProcesses++

			return nil, nil
		})

	src := &sourceStab{data: generateStringPayloads(maxNumOfWorkers * 2)}
	p := pipeline.New(pipeline.NewDynamicWorkerPool(proc, maxNumOfWorkers))

	go func() {
		err := p.Execute(context.TODO(), src, nil)
		c.Assert(err, check.IsNil)

		close(doneChan)
	}()

	// Wait for all workers to reach sync point. This means that each input
	// from the source is currently handled by a worker in parallel.
	for i := 0; i < maxNumOfWorkers; i++ {
		select {
		case <-syncChan:
		case <-time.After(10 * time.Second):
			c.Fatalf("timed out waiting for worker %d to reach sync point", i)
		}
	}

	// Allow workers / go-routines to proceed.
	close(rendezvousChan)

	// Wait for the pipeline to complete.
	select {
	case <-doneChan:
		close(syncChan)
	case <-time.After(10 * time.Second):
		c.Fatal("timed out waiting for pipeline to complete")
	}

	c.Assert(numOfExecutedProcesses, check.Equals, maxNumOfWorkers*2)
	assertAllPayloadProcessed(c, src.data...)
}

func (s *stageRunnerTestSuite) TestBroadcast(c *check.C) {
	numOfProcs := 3
	procs := make([]pipeline.Processor, numOfProcs)
	for i := 0; i < len(procs); i++ {
		procs[i] = generateMutatingProcessor(i)
	}

	src := &sourceStab{data: generateStringPayloads(1)}
	sink := new(sinkStab)
	p := pipeline.New(pipeline.NewBroadcastWorkerPool(procs...))

	err := p.Execute(context.TODO(), src, sink)
	c.Assert(err, check.IsNil)

	expectedData := []pipeline.Payload{
		&stringPayload{value: "0_0", isProcessed: true},
		&stringPayload{value: "0_1", isProcessed: true},
		&stringPayload{value: "0_2", isProcessed: true},
	}

	sort.Slice(expectedData, func(i, j int) bool {
		return expectedData[i].(*stringPayload).value < expectedData[j].(*stringPayload).value
	})

	sort.Slice(sink.data, func(i, j int) bool {
		return sink.data[i].(*stringPayload).value < sink.data[j].(*stringPayload).value
	})

	c.Assert(expectedData, check.DeepEquals, sink.data)
}

func generatePassThruProcessor() pipeline.Processor {
	return pipeline.ProcessorFunc(
		func(_ context.Context, p pipeline.Payload) (pipeline.Payload, error) {
			return p, nil
		})
}

func generateMutatingProcessor(index int) pipeline.Processor {
	return pipeline.ProcessorFunc(func(_ context.Context, p pipeline.Payload) (pipeline.Payload, error) {
		// Mutate payload in order to verify that each processor received a copy.
		payload := p.(*stringPayload)
		payload.value = fmt.Sprintf("%s_%d", payload.value, index)

		return payload, nil
	})
}
