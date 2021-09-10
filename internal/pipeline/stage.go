package pipeline

import (
	"context"
	"fmt"
	"sync"
)

// TODO: Refactor type constructors to return concrete struct types as
// opposed to interfaces

// Compile-time check for ensuring fifo implements StageRunner.
var _ StageRunner = (*fifo)(nil)

type fifo struct {
	proc Processor
}

// FIFO returns a StageRunner that processes incoming payloads in a first-in
// first-out fashion. Each input is passed to the specified processor and its
// output is emitted to the next stage.
func FIFO(proc Processor) StageRunner {
	return &fifo{
		proc: proc,
	}
}

// Run implements the StageRunner interface.
// It implements the payload processing loop for a single stage of the pipeline.
func (r *fifo) Run(ctx context.Context, params StageParams) {
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


// Compile-time check for ensuring fixedWorkerPool implements StageRunner.
var _ StageRunner = (*fixedWorkerPool)(nil)

// fixedWorkerPool spins up a preconfigured number of workers and distributes incoming
// payloads among them 
type fixedWorkerPool struct {
	fifos []StageRunner
}

// FixedWorkerPool returns a StageRunner that spins up a pool containing
// numOfWorkers to process incoming payloads in parallel and emit their outputs
// to the next stage.
func FixedWorkerPool(proc Processor, numOfWorkers int) StageRunner {
	if numOfWorkers <= 0 {
		panic("FixedWorkerPool: numOfWorkers must be > 0")
	}

	fifos := make([]StageRunner, numOfWorkers)
	for i := 0; i < numOfWorkers; i++ {
		fifos[i] = FIFO(proc)
	}

	return &fixedWorkerPool{
		fifos: fifos,
	}
}

// Run implements the StageRunner interface.
func (r *fixedWorkerPool) Run(ctx context.Context, params StageParams) {
	var wg sync.WaitGroup

	// Spin up each worker in the pool and wait for them to complete / return.
	for i := 0; i < len(r.fifos); i++ {
		wg.Add(1)

		go func(fifoIndex int) {
			r.fifos[fifoIndex].Run(ctx, params)

			wg.Done()
		}(i)
	}

	// Wait for all running worker goroutines to complete / return before allowing
	// the Run method to return.
	wg.Wait()
}


// Compile-time check for ensuring dynamicWorkerPool implements StageRunner.
var _ StageRunner = (*dynamicWorkerPool)(nil)

type dynamicWorkerPool struct {
	proc Processor
	tokenPool chan struct{}
}

// DynamicWorkerPool returns a StageRunner that maintains a dynamic worker pool
// that can scale up to maxWorkers for processing incoming inputs in parallel
// and emitting their outputs to the next stage.
func DynamicWorkerPool(proc Processor, maxNumOfWorkers int) StageRunner {
	if maxNumOfWorkers <= 0 {
		panic("DynamicWorkerPool: maxNumOfWorkers must be > 0")
	}

	tokenPool := make(chan struct{}, maxNumOfWorkers)
	for i := 0; i < maxNumOfWorkers; i++ {
		tokenPool <- struct{}{}
	}

	return &dynamicWorkerPool{
		proc: proc,
		tokenPool: tokenPool,
	}
}

func (r *dynamicWorkerPool) Run(ctx context.Context, params StageParams) {
	stop:
		for {
			select {
			case <-ctx.Done():
				break stop
			case payloadIn, ok := <-params.Input():
				if !ok {
					break stop
				}

				var token struct{}
				select {
				case token = <-r.tokenPool:
				case <-ctx.Done():
					break stop
				}

				go func(payloadIn Payload, token struct{}) {
					defer func() {
						r.tokenPool <- token
					}()

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

						return // Discard the payload.
					}

					// Output processed data.
					select {
					case params.Output() <- payloadOut:
					case <-ctx.Done():
					}

				}(payloadIn, token)
			}
		}

		// Wait for all workers to exit by trying to empty the token pool.
		for i := 0; i < cap(r.tokenPool); i++ {
			<-r.tokenPool
		}
}

