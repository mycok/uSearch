package pipeline

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

