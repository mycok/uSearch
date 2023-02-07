package pipeline

// Static and compile-time check to ensure the stageParams type implements
// StageParams interface.
var _ StageParams = (*stageParams)(nil)

type stageParams struct {
	stage   int
	inChan  <-chan Payload
	outChan chan<- Payload
	errChan chan<- error
}

func (p *stageParams) StageIndex() int {
	return p.stage
}

func (p *stageParams) Input() <-chan Payload {
	return p.inChan
}

func (p *stageParams) Output() chan<- Payload {
	return p.outChan
}

func (p *stageParams) Error() chan<- error {
	return p.errChan
}
