package control

import "sync"

// BlurParams — Gaussian Blur parameters.
type BlurParams struct {
	Ksize int // the size of the blur core (must be odd: 3, 5, 7...)
}

// EdgeParams — parameters of the Canny boundary detector.
type EdgeParams struct {
	Threshold1 float32 // lower sensitivity threshold
	Threshold2 float32 // upper sensitivity threshold
}

// BrightnessContrastParams — brightness and contrast parameters.
type BrightnessContrastParams struct {
	Alpha float64 // contrast ratio
	Beta  float64 // brightness shift
}

// PipelineParams is a common set of parameters for all pipeline stages.
type PipelineParams struct {
	mu                 sync.RWMutex
	Blur               BlurParams
	Edge               EdgeParams
	BrightnessContrast BrightnessContrastParams
}

func (p *PipelineParams) RLock()   { p.mu.RLock() }
func (p *PipelineParams) RUnlock() { p.mu.RUnlock() }
func (p *PipelineParams) Lock()    { p.mu.Lock() }
func (p *PipelineParams) Unlock()  { p.mu.Unlock() }

// NewPipelineParams creates default pipeline parameters.
func NewPipelineParams() *PipelineParams {
	return &PipelineParams{
		BrightnessContrast: BrightnessContrastParams{
			Alpha: 1.0,
			Beta:  0,
		},
		Blur: BlurParams{
			Ksize: 5,
		},
		Edge: EdgeParams{
			Threshold1: 50,
			Threshold2: 150,
		},
	}
}
