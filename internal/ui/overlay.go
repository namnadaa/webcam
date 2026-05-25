package ui

import (
	"time"
	"webcam/internal/control"
	"webcam/internal/pipeline"

	"gocv.io/x/gocv"
)

// RenderOverlay draws all UI layers on frame.
func RenderOverlay(
	img *gocv.Mat,
	state State,
	params *control.PipelineParams,
	stages []pipeline.StageToggle,
	fps float64,
	width, height int,
	cameraName string,
	frameTime time.Time,
) {
	if state.ShowMenu {
		DrawMenu(img)
	}

	if state.ShowStats {
		latency := time.Since(frameTime).Milliseconds()
		DrawStats(img, fps, latency, width, height, cameraName)
	}

	if state.ShowSettings {
		DrawSettings(img, params)
	}

	if state.ShowStatuses {
		DrawStatuses(img, stages)
	}

	if state.ShowControls {
		DrawControls(img)
	}
}
