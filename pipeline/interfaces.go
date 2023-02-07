package pipeline

import "context"

// Source should be implemented by types that generate Payload instances that
// can be used as inputs for a Pipeline instance.
type Source interface {
	// Next loads the next available payload from the source and returns true.
	// When no more payloads are available or an error occurs, calls to Next
	// return false.
	Next(context.Context) bool

	// Payload returns the current payload to be processed.
	Payload() Payload

	// Error returns the last error encountered by the source.
	Error() error
}

// Payload should be implemented by types that can serve as payloads for the pipeline.
type Payload interface {
	// Clone returns a deep-copy of the original payload.
	Clone() Payload

	// MarkAsProcessed is invoked by the stage / stage runner when the payload
	// either reaches the pipeline sink or it gets discarded by one of the
	// pipeline stages.
	MarkAsProcessed()
}

// Processor should be implemented by types that process payloads for a
// pipeline stage.
type Processor interface {
	// Process may transform the payload and return the transformed payload that
	// may be forwarded to the next stage in the pipeline. Processors may also
	// opt to prevent the payload from reaching the next stage of the pipeline by
	// returning a nil payload value instead. for example, if the payload is
	// malformed or doesn't meet the required conditions.
	Process(context.Context, Payload) (Payload, error)
}

// ProcessorFunc serves as an adapter that allows the use of normal functions
// as processor instances. If f is a function with the appropriate signature,
// ProcessorFunc(f) is a processor that calls f.
type ProcessorFunc func(context.Context, Payload) (Payload, error)

// Process calls f(ctx, p).
func (f ProcessorFunc) Process(ctx context.Context, p Payload) (Payload, error) {
	return f(ctx, p)
}

// StageRunner should be implemented by types that can be strung together
// to form a multi-stage pipeline.
type StageRunner interface {
	// Run receives the stage inputs / params from the pipeline which include
	// an input, output, and error channels and stage index that represents
	// the current stage position in the pipeline.

	// Calls to Run are expected to block until:
	// - the stage input channel is closed OR
	// - the provided context expires OR
	// - an error occurs while processing payloads.
	Run(context.Context, StageParams)
}

// StageParams should be implemented by types serve as inputs to a pipeline
// stage.
type StageParams interface {
	// StageIndex returns the current position of this stage in the pipeline.
	StageIndex() int

	// Input returns a read-only channel for reading the input payloads
	// for a stage.
	Input() <-chan Payload

	// Output returns a write-only channel for writing the output payloads
	// for a stage.
	Output() chan<- Payload

	// Error returns a write-only channel for writing errors that are
	// encountered by a stage while processing payloads.
	Error() chan<- error
}

// Sink should be implemented by types that serve as the last part of the
// pipeline.
type Sink interface {
	// Consume processes a payload instance that has been emitted out of
	// a pipeline.
	Consume(context.Context, Payload) error
}
