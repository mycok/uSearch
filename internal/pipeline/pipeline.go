package pipeline

import (
	"context"
	"fmt"
	"sync"

	"github.com/hashicorp/go-multierror"
)

// Compile-time check for ensuring workerParams implements StageParams.
var _ StageParams = (*workerParams)(nil)
 
type workerParams struct {
	stage int
	inCh <-chan Payload
	outCh chan<- Payload
	errCh chan<- error
}

func (p *workerParams) StageIndex() int {
	return p.stage
}

func (p *workerParams) Input() <-chan Payload {
	return p.inCh
}

func (p *workerParams) Output() chan<- Payload {
	return p.outCh
}

func (p *workerParams) Error() chan<- error {
	return p.errCh
}

// Pipeline implements a modular, multi-stage pipeline. Each pipeline is
// constructed out of an input source, an output sink and zero or more
// processing stages.
type Pipeline struct {
	stages []StageRunner
}

// New returns a new pipeline instance where input payloads will traverse each
// one of the specified stages.
func New(stages ...StageRunner) *Pipeline {
	return &Pipeline{
		stages: stages,
	}
}

// Process reads the contents of the specified source, sends them through the
// various stages of the pipeline and directs the results to the specified sink
// and returns back any errors that may have occurred.
//
// Calls to Process block until:
//  - all data from the source has been processed OR
//  - an error occurs OR
//  - the supplied context expires
//
// It is safe to call Process concurrently with different sources and sinks.
func (p *Pipeline) Process(ctx context.Context, src Source, sink Sink) error {
	var wg sync.WaitGroup
	processCtx, ctxCancelFn := context.WithCancel(ctx)

	// Allocate channels for wiring together the source, the pipeline stages
	// and the output sink. The output of the i_th stage is used as an input
	// for the i+1_th stage. We need to allocate one extra channel than the
	// number of stages so we can also wire the source/sink.
	stageChs := make([]chan Payload, len(p.stages) + 1)
	errCh := make(chan error, len(p.stages) + 2)

	for i := 0; i < len(stageChs); i++ {
		stageChs[i] = make(chan Payload)
	}

	// Start a worker / go-routine for each stage.
	for i := 0; i < len(p.stages); i++ {
		wg.Add(1)

		go func(stageIndex int) {
			p.stages[stageIndex].Run(processCtx, &workerParams{
				stage: stageIndex,
				inCh: stageChs[stageIndex],
				outCh: stageChs[stageIndex+1],
				errCh: errCh,
			})

			// Signal the next stage that no more data is available by closing the output channel.
			close(stageChs[stageIndex+1])

			wg.Done()
		}(i)
	}

	// Start source and sink workers / go-routines.
	wg.Add(2)

	go func() {
		sourceWorker(processCtx, src, stageChs[0], errCh)

		// Signal the next stage that no more data is available by closing the output channel.
		close(stageChs[0])

		wg.Done()
	}()

	go func() {
		sinkWorker(processCtx, sink, stageChs[len(stageChs) - 1], errCh)

		wg.Done()
	}()

	// Close the error channel once all workers exit / return.
	go func() {
		wg.Wait()

		close(errCh)
		ctxCancelFn()
	}()

	// Collect any emitted errors and wrap them in a multi-error.
	var err error
	for processErr := range errCh {
		err = multierror.Append(err, processErr)

		ctxCancelFn()
	}

	return err
}

// sourceWorker implements a worker that reads Payload instances from a Source
// and pushes them to an output channel that is used as input for the first
// stage of the pipeline.
func sourceWorker(ctx context.Context, src Source, outCh chan<- Payload, errCh chan<- error) {
	// Retrieve the payload from the source.
	for src.Next(ctx) {
		payload := src.Payload()

		select {
		case outCh <- payload:
		case <-ctx.Done(): // Ask context to close / shutdown
			return
		}
	}

	// Check for errors.
	if err := src.Error(); err != nil {
		wrappedErr := fmt.Errorf("pipeline source: %w", err)
		shouldEmitError(wrappedErr, errCh)
	}
}

// sinkWorker implements a worker that reads Payload instances from an input
// channel (the output of the last pipeline stage) and passes them to the
// provided sink.
func sinkWorker(ctx context.Context, sink Sink, inCh <-chan Payload, errCh chan<- error) {
	for {
		select {
		case payload, ok := <-inCh:
			if !ok {
				return
			}

			if err := sink.Consume(ctx, payload); err != nil {
				wrappedErr := fmt.Errorf("pipeline sink: %w", err)
				shouldEmitError(wrappedErr, errCh)

				return
			}

			payload.MarkAsProcessed()

		case <-ctx.Done(): // Ask context to close / shutdown
			return
		}
	}
}

// shouldEmitError attempts to queue err to a buffered error channel. If the
// channel is full, the error is dropped.
func shouldEmitError(err error, errCh chan<- error) {
	select {
	case errCh <- err: // Error emitted
	default: // errCh is full with other errors and the new err is dropped.
	}
}


