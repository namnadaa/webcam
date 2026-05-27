package pipeline

import (
	"sync"
	"time"
	"webcam/internal/control"

	"gocv.io/x/gocv"
)

// Frame represents a single video frame with timestamp.
type Frame struct {
	Img  gocv.Mat
	Time time.Time
}

// Close releases frame resources.
func (f *Frame) Close() {
	f.Img.Close()
}

// StageToggle — pipeline element that can be turned on/off
type StageToggle struct {
	Enabled bool
	Build   func(params *control.PipelineParams) StageFunc
}

// StageFunc defines a processing stage function type.
type StageFunc func(prev <-chan Frame, done <-chan struct{}, wg *sync.WaitGroup) <-chan Frame

// NewStages creates default pipeline stages.
func NewStages() []StageToggle {
	return []StageToggle{
		{
			Enabled: false,
			Build: func(p *control.PipelineParams) StageFunc {
				return BrightnessContrastStage(&p.BrightnessContrast)
			},
		},
		{
			Enabled: false,
			Build: func(p *control.PipelineParams) StageFunc {
				return BlurStage(&p.Blur)
			},
		},
		{
			Enabled: false,
			Build: func(p *control.PipelineParams) StageFunc {
				return EdgeStage(&p.Edge)
			},
		},
		{
			Enabled: false,
			Build: func(p *control.PipelineParams) StageFunc {
				return GrayStage()
			},
		},
		{
			Enabled: false,
			Build: func(p *control.PipelineParams) StageFunc {
				return SharpenStage()
			},
		},
	}
}
