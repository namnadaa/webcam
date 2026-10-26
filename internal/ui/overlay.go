package ui

import (
	"time"
	"webcam/internal/control"
	"webcam/internal/pipeline"

	"gocv.io/x/gocv"
)

// RenderOverlay draws all UI layers on frame, scaled to the frame height.
// Right-side panels are stacked one under another; hidden panels take no space.
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
	l := newLayout(img)

	if state.ShowMenu {
		drawMenu(img, l)
	}

	y := l.top

	if state.ShowStats {
		latency := time.Since(frameTime).Milliseconds()
		y = drawStats(img, l, y, fps, latency, width, height, format, cameraName)
	}

	if state.ShowSettings {
		y = drawSettings(img, l, y, params)
	}

	if state.ShowStatuses {
		y = drawStatuses(img, l, y, stages)
	}

	if state.ShowControls {
		drawControls(img, l, y)
	}

	drawExperiment(img, l, experiment)

	if time.Now().Before(state.Notification.Until) {
		drawNotification(img, l, state.Notification.Message)
	}
}
