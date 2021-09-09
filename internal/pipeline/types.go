package pipeline

import "context"

// Source should be implemented by types that generate Payload instances which can be
// used as inputs to a Pipeline instance.
type Source interface {
	// Next fetches the next available payload from the source and returns true.
	// If no more payloads are available or an error occurs, calls to Next return false.
	Next(context.Context) bool

	// Payload returns the next payload to be processed.
	Payload() Payload

	// Error returns the last error observed by the source.
	Error() error
}

// Sink should be implemented by types that serve as the last part of the pipeline.
type Sink interface {
	// Consume processes a Payload instance that has been emitted out of
	// a Pipeline instance.
	Consume(context.Context, Payload) error
}

// Payload should be implemented by types / values that can be sent through the pipeline.
type Payload interface {
	// Clone returns a deep-copy of the original payload
	Clone() Payload

	// MarkAsProcessed is invoked by the pipeline when the Payload either
	// reaches the pipeline sink or it gets discarded by one of the
	// pipeline stages.
	MarkAsProcessed()
}

// StageParams encapsulates the information required for executing a pipeline
// stage. The pipeline passes a StageParams instance to the Run() method of
// each stage.
type StageParams interface {
	// StageIndex returns the current position of this stage in the pipeline.
	StageIndex() int

	// Input returns a read-only channel for reading the input payloads for a stage.
	Input() <-chan Payload

	// Output returns a write-only channel for writing the output payloads for a stage.
	Output() chan<- Payload

	// Error returns a write-only channel for writing errors that were encountered by
	// a stage while processing payloads.
	Error() chan<- error
}

// Processor is implemented by types that can process Payloads as part of a
// pipeline stage.
type Processor interface {
	// Process operates on the input payload and returns back a new payload
	// to be forwarded to the next pipeline stage. Processors may also opt
	// to prevent the payload from reaching the rest of the pipeline by
	// returning a nil payload value instead.
	Process(context.Context, Payload) (Payload, error)
}

// ProcessorFunc is an adapter to allow the use of plain functions as Processor
// instances. If f is a function with the appropriate signature, ProcessorFunc(f)
// is a Processor that calls f.
type ProcessorFunc func(context.Context, Payload) (Payload, error)

// Process calls f(ctx, p).
func (f ProcessorFunc) Process(ctx context.Context, p Payload) (Payload, error) {
	return f(ctx, p)
}

// StageRunner should be implemented by types that can be strung together to form a
// multi-stage pipeline.
type StageRunner interface {
	// Run implements the processing logic for the stage by reading
	// incoming Payloads from an input channel, processing them and
	// outputting the results to an output channel.
	//
	// Calls to Run are expected to block until:
	// - the stage input channel is closed OR
	// - the provided context expires OR
	// - an error occurs while processing payloads.
	Run(context.Context, StageParams)
}
