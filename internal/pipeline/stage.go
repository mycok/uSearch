package pipeline

import (
	"context"
	"fmt"
)

// Compile-time check for ensuring fifo implements StageRunner.
var _ StageRunner = (*fifo)(nil)

type fifo struct {
	proc Processor
}

// FIFO returns a StageRunner that processes incoming payloads in a first-in
// first-out fashion. Each input is passed to the specified processor and its
// output is emitted to the next stage.
func FIFO(proc Processor) StageRunner {
	return fifo{
		proc: proc,
	}
}

// Run implements the StageRunner interface.
// It implements the payload processing loop for a single stage of the pipeline.
func (r fifo) Run(ctx context.Context, params StageParams) {
	for {
		select {
		case <-ctx.Done(): // Ask context to close / shutdown.
			return
		case payloadIn, ok := <-params.Input():
			if !ok {
				return // No more data available.
			}

			payloadOut, err := r.proc.Process(ctx, payloadIn)
			if err != nil {
				wrappedError := fmt.Errorf("pipeline stage %d: %w", params.StageIndex(), err)
				shouldEmitError(wrappedError, params.Error())

				return
			}

			// If the processor did not output a payload for the
			// next stage there is nothing we need to do.
			if payloadOut == nil {
				payloadIn.MarkAsProcessed()

				continue
			}

			// Output processed data.
			select {
			case params.Output() <- payloadOut:
			case <-ctx.Done():
				return
			} 
		}
	}
}