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
// It implements the payload processing infinite loop for a single stage of the pipeline.
func (r *fifo) Run(ctx context.Context, params StageParams) {
	for {
		select {
		case <-ctx.Done(): // Ask context to close / shutdown.
			return
		case payloadIn, ok := <-params.Input():
			if !ok {
				return // No more data available, channel is closed.
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

				continue // Start a new iteration.
			}

			// Output processed data.
			// Note: the code in the select block blocks until either context is cancelled by
			// the user or the output payload is successfully written to the output channel in
			// which case a new iteration starts.
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
// It implements the payload processing for-loop that spins up a fixed number of workers / go-routines 
// for each fifo of []fifos.
func (r *fixedWorkerPool) Run(ctx context.Context, params StageParams) {
	var wg sync.WaitGroup

	// Spin up each worker in the pool and wait for them to complete / return.

	// Note: each worker reads from the same input channel and writes to the same output channel.
	// payloads are distributed to whichever worker is free and ready to read and process a new payload.
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

// Run implements the StageRunner interface.
// It implements the payload processing infinite loop that dynamically spins up workers / go-routines
// for each available token.
func (r *dynamicWorkerPool) Run(ctx context.Context, params StageParams) {

	// This infinite for loop keeps trying to read from [params.input()] channel.
	// if a read is successful, the select statement blocks until a token is read from the tokenPool.
	// for every token read, a go-routine is started that processes and writes the output payload to
	// the output channel.
	// When the maxNumOfWorkers is reached, the main loop blocks until a new token is available. This
	// only happens after any of the initial maxNumOfWorkers go-routines completes / returns.
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

	// Wait for all workers to exit by trying to empty the token pool. since spinning up of new
	// workers depends on the presence of a token, draining the token pool of all available tokens
	// ensures that no more workers / go-routines are spun up and that all active workers / go-routines exit.

	// Note: This loop runs after the main [stop] infinite loop has returned either due to
	// context cancellation or when the [params.input] channel is closed.
	for i := 0; i < cap(r.tokenPool); i++ {
		<-r.tokenPool
	}
}


// Compile-time check for ensuring fixedWorkerPool implements StageRunner.
var _ StageRunner = (*broadcast)(nil)

type broadcast struct {
	fifos []StageRunner
}

// Broadcast returns a StageRunner that passes a copy of each incoming payload
// to all specified processors and emits their outputs to the next stage.
func Broadcast(procs ...Processor) StageRunner {
	if len(procs) == 0 {
		panic("Broadcast: at least one processor must be specified")
	}

	fifos := make([]StageRunner, len(procs))
	for i, p := range procs {
		fifos[i] = FIFO(p)
	}

	return &broadcast{
		fifos: fifos,
	}
}

func (r *broadcast) Run(ctx context.Context, params StageParams) {
	var (
		wg sync.WaitGroup
		inChs = make([]chan Payload, len(r.fifos))
	)

	// Start each FIFO in a go-routine. Each FIFO gets its own dedicated
	// input channel and the shared output channel passed to Run.
	for i := 0; i < len(r.fifos); i++ {
		wg.Add(1)

		inChs[i] = make(chan Payload)

		go func(fifoIndex int) {
			fifoParams := &workerParams{
				stage: params.StageIndex(),
				inCh:  inChs[fifoIndex],
				outCh: params.Output(),
				errCh: params.Error(),
			}
			
			r.fifos[fifoIndex].Run(ctx, fifoParams)

			wg.Done()
		}(i)
	}

	// This is the main loop and it runs constantly until the [params.input] channel
	// is closed or the context gets cancelled.
	done:
		for {
			// Read incoming payloads then pass them to each FIFO instance.
			// The select block of code blocks until either the payload is read from
			// the input channel or the context gets cancelled.
			select {
			case <-ctx.Done():
				break done
			case payload, ok := <-params.Input():
				if !ok {
					break done
				}

				// This loop writes payloads to each dedicated FIFO input channel.
				// After writing payloads to all FIFO input channels, a new iteration
				// of the main begins.
				for i := len(r.fifos) - 1; i >= 0; i-- {
					// Since each FIFO might modify the payload, in order
					// avoid data races we need to make a copy of
					// the payload for all FIFOs except the first.
					var fifoPayload = payload
					if i != 0 {
						fifoPayload = payload.Clone()
					}

					select {
					case <-ctx.Done():
						break done
					case inChs[i] <- fifoPayload:
					}

				}
			}
		}

	// This loop runs after the main loop has exited.
	// Signal each of the fifo stage runner instances to shut down by closing their 
	// dedicated input channels.
	for _, ch := range inChs {
		close(ch)
	}

	// Wait for all the go-routines to exit before allowing the Run method to return.
	wg.Wait()
}

