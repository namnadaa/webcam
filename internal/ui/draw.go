package ui

import (
	"fmt"
	"image"
	"image/color"
	"webcam/internal/control"
	"webcam/internal/helpers"
	"webcam/internal/pipeline"

	"gocv.io/x/gocv"
)

const (
	textScale     = 1.2
	textThickness = 1
	outlineSize   = 3

	leftPadding = 10
	topPadding  = 20

	lineHeight = 20

	statsY    = 20
	settingsY = 120
	statusesY = 240
	controlsY = 360
)

// drawText renders outlined text on an image for better visibility.
func drawText(img *gocv.Mat, text string, x, y int, clr color.RGBA) {
	gocv.PutText(
		img,
		text,
		image.Pt(x, y),
		gocv.FontHersheyPlain,
		textScale,
		color.RGBA{0, 0, 0, 0},
		outlineSize,
	)

	gocv.PutText(
		img,
		text,
		image.Pt(x, y),
		gocv.FontHersheyPlain,
		textScale,
		clr,
		textThickness,
	)
}

// drawRightText renders right-aligned outlined text.
func drawRightText(img *gocv.Mat, text string, y int, clr color.RGBA) {
	size := gocv.GetTextSize(text, gocv.FontHersheyPlain, textScale, textThickness)
	x := img.Cols() - size.X - leftPadding

	drawText(img, text, x, y, clr)
}

// DrawMenu renders the interactive control menu overlay.
func DrawMenu(img *gocv.Mat) {
	menu := []string{
		"======== MENU ========",
		"",
		"[1] Toggle Brightness/Contrast",
		"[2] Toggle Blur",
		"[3] Toggle Edge",
		"[4] Toggle Gray",
		"[5] Toggle Sharpen",
		"",
		"[T] Toggle Statistics",
		"[P] Toggle Parameters",
		"[F] Toggle Filters",
		"[C] Toggle Controls",
		"",
		"[X] Screenshot",
		"[R] Video Recording",
		"",
		"[TAB] Close Menu",
		"[ESC] Camera Menu",
	}

	x := leftPadding
	y := topPadding

	for _, line := range menu {
		if line == "" {
			y += lineHeight
			continue
		}

		drawText(img, line, x, y, color.RGBA{255, 255, 255, 0})
		y += lineHeight
	}
}

// DrawStats renders performance statistics.
func DrawStats(img *gocv.Mat, fps float64, latency int64, width int, height int, cameraName string) {
	stats := []string{
		fmt.Sprintf("FPS: %.2f", fps),
		fmt.Sprintf("Latency: %d ms", latency),
		fmt.Sprintf("Resolution: %dx%d", width, height),
		fmt.Sprintf("Camera: %s", cameraName),
	}

	y := statsY

	for _, line := range stats {
		drawRightText(img, line, y, color.RGBA{255, 165, 0, 0})
		y += lineHeight
	}
}

// DrawSettings renders current filter settings.
func DrawSettings(img *gocv.Mat, params *control.PipelineParams) {
	settings := []string{
		fmt.Sprintf("Brightness: %.0f", params.BrightnessContrast.Beta),
		fmt.Sprintf("Contrast: %.2f", params.BrightnessContrast.Alpha),
		fmt.Sprintf("Blur Kernel: %d", params.Blur.Ksize),
		fmt.Sprintf("Edge T1: %.0f", params.Edge.Threshold1),
		fmt.Sprintf("Edge T2: %.0f", params.Edge.Threshold2),
	}

	y := settingsY

	for _, line := range settings {
		drawRightText(img, line, y, color.RGBA{0, 255, 255, 0})
		y += lineHeight
	}
}

// DrawStatuses renders filter enable statuses.
func DrawStatuses(img *gocv.Mat, stages []pipeline.StageToggle) {
	statuses := []string{
		fmt.Sprintf("Brightness/Contrast [%s]", helpers.OnOff(stages[0].Enabled)),
		fmt.Sprintf("Blur [%s]", helpers.OnOff(stages[1].Enabled)),
		fmt.Sprintf("Edge [%s]", helpers.OnOff(stages[2].Enabled)),
		fmt.Sprintf("Gray [%s]", helpers.OnOff(stages[3].Enabled)),
		fmt.Sprintf("Sharpen [%s]", helpers.OnOff(stages[4].Enabled)),
	}

	y := statusesY

	for _, line := range statuses {
		drawRightText(img, line, y, color.RGBA{0, 255, 0, 0})
		y += lineHeight
	}
}

// DrawControls renders camera control bindings.
func DrawControls(img *gocv.Mat) {
	controls := []string{
		"Brightness: W / S",
		"Contrast:   A / D",
		"Blur Size:  E / Q",
		"Edge T1:    J / K",
		"Edge T2:    N / M",
	}

	y := controlsY

	for _, line := range controls {
		drawRightText(img, line, y, color.RGBA{200, 200, 200, 0})
		y += lineHeight
	}
}

// DrawNotification renders notification text on frame.
func DrawNotification(img *gocv.Mat, text string) {
	x := leftPadding
	y := img.Rows() - topPadding

	drawText(img, text, x, y, color.RGBA{255, 0, 0, 0})
}
