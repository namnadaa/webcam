package pipeline

import (
	"time"

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

// StageFunc defines a processing stage function type.
type StageFunc func(prev <-chan Frame, done <-chan struct{}) <-chan Frame
