package pipeline

import (
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
type StageFunc func(prev <-chan Frame, done <-chan struct{}) <-chan Frame
