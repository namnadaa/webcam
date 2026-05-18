package pipeline

import (
	"sync"
	"webcam/internal/camera"
	"webcam/internal/control"
)

// BuildPipeline dynamically builds a pipeline based on current parameters and enabled stages.
func BuildPipeline(cam *camera.Camera, stages []StageToggle, params *control.PipelineParams, done <-chan struct{}, wg *sync.WaitGroup) <-chan Frame {
	frames := ReadStage(cam, done, wg)

	for _, st := range stages {
		if st.Enabled {
			frames = st.Build(params)(frames, done, wg)
		}
	}

	frames = ToBGRStage()(frames, done, wg)
	return frames
}
