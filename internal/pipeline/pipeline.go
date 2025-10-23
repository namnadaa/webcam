package pipeline

// Pipeline represents a chain of connected processing stages.
type Pipelile struct {
	stages []StageFunc
	done   <-chan struct{}
}

// New creates a new Pipeline with the given cancellation channel
// and ordered list of processing stages.
func New(done <-chan struct{}, stages ...StageFunc) *Pipelile {
	return &Pipelile{
		stages: stages,
		done:   done,
	}
}

// Run connects all pipeline stages together and returns
// the output channel of the last stage.
func (p *Pipelile) Run(source <-chan Frame) <-chan Frame {
	out := source
	for _, stage := range p.stages {
		out = p.runStageFrame(stage, out)
	}
	return out
}

// runStageFrame connects a single stage to its input channel
// and returns the resulting output channel.
func (p *Pipelile) runStageFrame(stage StageFunc, sourceChan <-chan Frame) <-chan Frame {
	return stage(sourceChan, p.done)
}
