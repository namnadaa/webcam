package pipeline

import (
	"webcam/internal/camera"
	"webcam/internal/control"
)

// BuildPipeline dynamically builds a pipeline based on current parameters and enabled stages.
func BuildPipeline(cam *camera.Camera, stages []StageToggle, params *control.PipelineParams, done chan struct{}) <-chan Frame {
	frames := ReadStage(cam, done)

	for _, st := range stages {
		if st.Enabled {
			frames = st.Build(params)(frames, done)
		}
	}

	return frames
}
