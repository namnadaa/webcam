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
	format string,
	cameraName string,
	frameTime time.Time,
	experiment ExperimentInfo,
) {
	if state.ShowMenu {
		DrawMenu(img)
	}

	if state.ShowStats {
		latency := time.Since(frameTime).Milliseconds()
		DrawStats(img, fps, latency, width, height, format, cameraName)
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

	DrawExperiment(img, experiment)

	if time.Now().Before(state.Notification.Until) {
		DrawNotification(img, state.Notification.Message)
	}
}
