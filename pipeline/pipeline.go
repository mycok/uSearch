/*
	pipeline package provides an implementation of an asynchronous pipeline
	abstraction with a set of application pipeline features and functionality
	exposed through a synchronous API.
		Features and Functionality:
			-

		Requirements:
			- The client / user should provide an input source that satisfies
			  the [pipeline.Source] interface.
			- The client / user should provide an output sink that satisfies
			  the [pipeline.Sink] interface.
			- The client / user may use the available package stage runner
			  concrete implementations by creating instances or embedding them
			  within his user defined stage runner types.
			- The client / user may define stage runner types that satisfy
			  [pipeline.StageRunner] interface.
*/

package pipeline

import (
	"context"
	"fmt"
	"sync"

	"github.com/hashicorp/go-multierror"
)

// Pipeline provide modular, multi-stage pipeline functionality. Each pipeline
// is built out of an input source, an output sink and zero or more
// processing stages / stage runners.
type Pipeline struct {
	stages []StageRunner
}

// New returns a pointer to a pipeline instance.
func New(stages ...StageRunner) *Pipeline {
	return &Pipeline{stages}
}

// Execute reads the contents of the specified source, sends them through the
// various stages of the pipeline and directs the results to the specified sink
// and returns back any errors that may have occurred.
//
// Calls to execute block until:
//   - all data from the source has been processed or discarded.
//   - an error is encountered from any of the pipeline components including
//     stage runners and their user defined processor functions.
//   - the supplied context is cancelled.
//
// It is safe to call execute concurrently with different sources and sinks.
func (p *Pipeline) Execute(ctx context.Context, src Source, sink Sink) error {
	var wg sync.WaitGroup
	executionCtx, cancel := context.WithCancel(ctx)

	// Allocate channels for wiring together the source, the pipeline stages
	// and the output sink. The output of the i_th stage is used as an input
	// for the i+1_th stage. We need to allocate one extra channel than the
	// number of stages which should be used to connect the source and sink
	// components in case no stages / stage runners are provided. (pass through).
	stageChans := make([]chan Payload, len(p.stages)+1)
	// Initialize all stage channels as empty channels of payload type.
	for i := 0; i < len(stageChans); i++ {
		stageChans[i] = make(chan Payload)
	}

	// Allocate a buffered channel to accommodate errors from all pipeline
	// components including the source, sink and stages / stage runners.
	errChan := make(chan error, len(p.stages)+2)

	// Launch a worker / goroutine for each pipeline stage / stage runner.
	for i := 0; i < len(p.stages); i++ {
		wg.Add(1)

		go func(index int) {
			defer wg.Done()

			p.stages[index].Run(executionCtx, &stageParams{
				stage:   index,
				inChan:  stageChans[index],
				outChan: stageChans[index+1],
				errChan: errChan,
			})

			// Once the run method for a particular stage returns, we signal
			// the next stage that no more data is available by closing the
			// output channel.

			// Note: Run method for a particular stage will only return if it's
			// input channel has been closed by the previous stage or closed by
			// the sourceWorker in case it's the first stage to run in the
			// pipeline or when the stage's process method returns an error
			// that causes it to return prematurely. It's also important to
			// note that any return / exit from any stages's Run method will
			// trigger a chain of exits for all the other stages which will in
			// turn cause the pipeline to exit.
			close(stageChans[index+1])
		}(i)
	}

	// Start source and sink workers / go-routines.
	wg.Add(2)

	go func() {
		sourceWorker(executionCtx, src, stageChans[0], errChan)

		// Once the sourceWorker's source runs out of data or ctx is cancelled,
		// sourceWorker returns and it signals the stage that reads from it's
		// channel [stageChs[0]] that no more data is available by closing that
		// channel. This will start a chain of channel closures from the
		// respective stages since there will be no more data to be read.
		close(stageChans[0])
		wg.Done()
	}()

	go func() {
		sinkWorker(executionCtx, sink, stageChans[len(stageChans)-1], errChan)
		wg.Done()
	}()

	// Ensure all workers / goroutines exit, close the error channel and cancel
	// the provided context.
	go func() {
		wg.Wait()

		close(errChan)
		cancel()
	}()

	var err error
	for stageErr := range errChan {
		err = multierror.Append(err, stageErr)

		// Cancel the provided context and trigger the shutdown of the entire
		// pipeline.
		cancel()
	}

	return err
}

// sourceWorker retrieves payload instances from a source object and sends them
// to an input channel that serves the payloads as input for the first
// stage / stage runner of the pipeline.
func sourceWorker(
	ctx context.Context, src Source,
	outChan chan<- Payload, errChan chan<- error) {

	for src.Next(ctx) {
		p := src.Payload()

		select {
		case <-ctx.Done():
			return
		case outChan <- p:
		}
	}

	if err := src.Error(); err != nil {
		wrappedErr := fmt.Errorf("pipeline source: %w", err)
		mayEmitError(wrappedErr, errChan)
	}
}

func sinkWorker(
	ctx context.Context, sink Sink,
	inChan <-chan Payload, errChan chan<- error,
) {
	for {
		select {
		case <-ctx.Done():
			return
		case payload, ok := <-inChan:
			if !ok {
				return // if channel is closed.
			}

			if err := sink.Consume(ctx, payload); err != nil {
				wrappedErr := fmt.Errorf("pipeline sink: %w", err)
				mayEmitError(wrappedErr, errChan)

				return
			}

			payload.MarkAsProcessed()
		}
	}
}

func mayEmitError(err error, errChan chan<- error) {
	select {
	case errChan <- err: // error is successfully written to the channel.
	default: // errChan is full of old errors and the new error is dropped.
	}
}
