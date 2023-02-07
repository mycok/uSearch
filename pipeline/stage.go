package pipeline

import (
	"context"
	"fmt"
	"sync"
)

// fifo uses a (first-in-first-out) payload dispatch strategy. it's most suited
// for cases where order of output is desired.
type fifo struct {
	proc Processor
}

// NewFIFO returns a StageRunner that processes incoming payloads in a first-in
// first-out fashion.
func NewFIFO(proc Processor) StageRunner {
	return fifo{proc}
}

// Run processes the provided payload, if the processing is successful the
// payload is dispatched to the next stage / stage runner by writing it to
// the provided params.Output() channel.
// In case an error occurs during payload processing, that error is returned by
// the processor function, wrapped and written to the params.Error() channel.
// Run blocks during it's execution and returns or exits if the provided context
// is cancelled, the input channel is closed or if an error is returned by the
// stage / stage runner processor function.
func (r fifo) Run(ctx context.Context, params StageParams) {
	for {
		select {
		case <-ctx.Done():
			return // context timeout or cancelled.
		case payloadIn, ok := <-params.Input():
			if !ok {
				return // input channel closed.
			}

			payloadOut, err := r.proc.Process(ctx, payloadIn)
			if err != nil {
				wrappedErr := fmt.Errorf(
					"pipeline stage %d: %w", params.StageIndex(), err,
				)
				mayEmitError(wrappedErr, params.Error())

				return
			}

			// For cases where the processor did not return a payload for the
			// next stage, we continue to read and process new payloads.
			if payloadOut == nil {
				payloadIn.MarkAsProcessed()

				continue // next iteration cycle.
			}

			select {
			case <-ctx.Done():
				return // context timeout or cancelled.
			case params.Output() <- payloadOut: // next iteration cycle.
			}
		}
	}
}

// fixedWorkerPool stage runner distributes incoming payloads among a constant
// number of workers / goroutines.
type fixedWorkerPool struct {
	fifos []StageRunner
}

// NewFixedWorkerPool returns a StageRunner that uses a pool of other
// fifo stage runners of a fixed number to process incoming payloads.
func NewFixedWorkerPool(proc Processor, numOfWorkers int) StageRunner {
	if numOfWorkers <= 0 {
		panic("FixedWorkerPool: numOfWorkers must be > 0")
	}

	fifos := make([]StageRunner, numOfWorkers)
	for i := 0; i < numOfWorkers; i++ {
		fifos[i] = NewFIFO(proc)
	}

	return fixedWorkerPool{fifos}
}

// Run uses a loop to launch a fixed number of workers / go-routines
// for each fifo runner instance from a list of stage runners.
func (r fixedWorkerPool) Run(ctx context.Context, params StageParams) {
	var wg sync.WaitGroup

	// Launch a worker / goroutine for each stage runner instance.
	// Note: each worker reads from the same input channel and writes
	// to the same output channel.
	// payloads are distributed to whichever worker is free and ready to
	//  read and process a new payload (load balancing).
	for i := 0; i < len(r.fifos); i++ {
		wg.Add(1)

		go func(index int) {
			r.fifos[index].Run(ctx, params)

			wg.Done()
		}(i)
	}

	// Wait for all running worker goroutines to complete / exit,
	wg.Wait()
}

// dynamicWorkerPool stage runner uses a token pool to dynamically determine the
// number of workers / goroutines to be launched and distributes incoming
// payloads among them.
type dynamicWorkerPool struct {
	proc      Processor
	tokenPool chan struct{}
}

// NewDynamicWorkerPool uses a token pool to dynamically determine the number of
// workers / goroutines to be launched and distributes incoming payloads among
// them.
func NewDynamicWorkerPool(proc Processor, maxNumOfWorkers int) StageRunner {
	if maxNumOfWorkers <= 0 {
		panic("DynamicWorkerPool: maxNumOfWorkers must be > 0")
	}

	// Instantiate a buffered token pool channel.
	tokenPool := make(chan struct{}, maxNumOfWorkers)
	for i := 0; i < maxNumOfWorkers; i++ {
		tokenPool <- struct{}{}
	}

	return dynamicWorkerPool{
		proc:      proc,
		tokenPool: tokenPool,
	}
}

// Run uses an infinite loop that dynamically launches workers / go-routines
// for each available token.
func (r dynamicWorkerPool) Run(ctx context.Context, params StageParams) {
	// This infinite for loop keeps trying to read from [params.input()] channel.
	// if a read is successful, the select statement blocks until a token is
	// read from the tokenPool.
	// For every token read, a go-routine is started that processes and writes
	// the output payload to the output channel. When the maxNumOfWorkers is
	// reached, the main loop blocks until a new token is available. This only
	// happens after any of the initial maxNumOfWorkers go-routines completes.

	// Note: All workers / goroutines that are launched share the same output
	// channel. (params.Output())
outer:
	for {
		select {
		case <-ctx.Done():
			break outer // context timed-out or cancelled.
		case payloadIn, ok := <-params.Input():
			if !ok {
				break outer // input channel closed.
			}

			var token struct{}

			select {
			case <-ctx.Done():
				break outer
			case token = <-r.tokenPool:
			}

			// Launch a goroutine to process the payload.
			go func(p Payload, t struct{}) {
				defer func() {
					r.tokenPool <- token
				}()

				payloadOut, err := r.proc.Process(ctx, p)
				if err != nil {
					wrappedErr := fmt.Errorf(
						"pipeline stage %d: %w", params.StageIndex(), err,
					)
					mayEmitError(wrappedErr, params.Error())

					return // return from the goroutine.
				}

				// For cases where the processor did not return a payload
				// for the next stage, we discard the payload.
				if payloadOut == nil {
					payloadIn.MarkAsProcessed()

					return // Discard the payload
				}

				select {
				case <-ctx.Done(): // context timeout or cancelled.
				case params.Output() <- payloadOut:
				}
			}(payloadIn, token)
		}
	}

	// Wait for all workers to exit by trying to empty the token pool.
	// since launching of new workers depends on the presence of a token,
	// draining the token pool of all available tokens ensures that all active
	// workers / go-routines exit.

	// Note: This loop runs after the main [outer] infinite loop has returned
	// either due to context cancellation or when the [params.input] channel
	//  is closed. This is equivalent to using the sync.WaitGroup and calling
	// sync.WaitGroup.Wait().
	for i := 0; i < cap(r.tokenPool); i++ {
		<-r.tokenPool
	}
}

type broadcast struct {
	fifos []StageRunner
}

// NewBroadcastWorkerPool returns a StageRunner that clones and passes a copy of each
// incoming payload to all specified processors and emits their outputs to the
// next stage.
func NewBroadcastWorkerPool(procs ...Processor) StageRunner {
	if len(procs) == 0 {
		panic("Broadcast: at least one processor must be specified")
	}

	fifos := make([]StageRunner, len(procs))
	for i, p := range procs {
		fifos[i] = NewFIFO(p)
	}

	return broadcast{fifos}
}

// Run launches go-routines / workers for each fifo stage runner and passes it it's
// own input channel and a shared output channel.
func (r broadcast) Run(ctx context.Context, params StageParams) {
	var (
		wg      sync.WaitGroup
		inChans = make([]chan Payload, len(r.fifos))
	)

	// Start each FIFO stage runner in a goroutine. each stage runner gets its
	// own dedicated input channel and the shared output channel from the
	//  stageParams.
	for i := 0; i < len(r.fifos); i++ {
		wg.Add(1)

		inChans[i] = make(chan Payload)

		go func(index int) {
			defer wg.Done()

			fifoParams := &stageParams{
				stage:   params.StageIndex(),
				inChan:  inChans[index],
				outChan: params.Output(),
				errChan: params.Error(),
			}

			r.fifos[index].Run(ctx, fifoParams)
		}(i)
	}

	// Outer loop blocks until the [params.input] channel is closed or the
	//  context gets cancelled.
outer:
	for {
		// Read incoming payloads then pass them to each FIFO stage runner
		// instance. The select block of code blocks until either the payload
		// is read from the input channel or the context gets cancelled.
		select {
		case <-ctx.Done():
			break outer
		case payload, ok := <-params.Input():
			if !ok {
				break outer
			}

			// Write payloads to each dedicated FIFO input channel.
			// After writing payloads to all FIFO input channels, a new
			// iteration cycle of the outer loop begins.
			for i := len(r.fifos) - 1; i >= 0; i-- {
				// Since each FIFO might modify the payload, in order
				// avoid data races we need to make a copy of
				// the payload for all FIFOs except the first.
				fifoPayload := payload
				// Check if it's the FIFO stage runner instance.
				if i != 0 {
					fifoPayload = payload.Clone()
				}

				select {
				case <-ctx.Done():
					break outer // Stop reading payloads from params.Input chan.
				case inChans[i] <- fifoPayload:
				}
			}
		}
	}

	// Signal each of the fifo stage runner instances to shut down by closing their
	// dedicated input channels.
	for _, ch := range inChans {
		close(ch)
	}

	// Wait for all goroutines / workers to complete or exit.
	wg.Wait()
}
