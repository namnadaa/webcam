package ui

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"webcam/internal/control"
	"webcam/internal/helpers"
	"webcam/internal/pipeline"

	"gocv.io/x/gocv"
)

// Sizes at the reference frame height; everything is scaled by height/referenceHeight.
const (
	referenceHeight = 720
	minScale        = 0.5

	textScale     = 1.2
	textThickness = 1
	outlineSize   = 3

	leftPadding = 10
	topPadding  = 20
	lineHeight  = 20

	// lines kept free at the bottom for the experiment panel and the notification
	bottomLines = 5
)

// layout holds overlay sizes scaled to the frame height.
type layout struct {
	font      float64
	thickness int
	outline   int
	pad       int
	top       int
	line      int
}

// scaleFor returns the overlay scale for a frame height.
func scaleFor(rows int) float64 {
	return math.Max(minScale, float64(rows)/referenceHeight)
}

// newLayout computes overlay sizes for the frame.
func newLayout(img *gocv.Mat) layout {
	s := scaleFor(img.Rows())
	scaled := func(v float64) int { return int(math.Round(v * s)) }

	return layout{
		font:      textScale * s,
		thickness: max(1, scaled(textThickness)),
		outline:   max(2, scaled(outlineSize)),
		pad:       scaled(leftPadding),
		top:       scaled(topPadding),
		line:      scaled(lineHeight),
	}
}

// textWidth returns the rendered width of text in pixels.
func (l layout) textWidth(text string) int {
	return gocv.GetTextSize(text, gocv.FontHersheyPlain, l.font, l.thickness).X
}

// drawText renders outlined text on an image for better visibility.
func (l layout) drawText(img *gocv.Mat, text string, x, y int, clr color.RGBA) {
	gocv.PutText(img, text, image.Pt(x, y), gocv.FontHersheyPlain, l.font, color.RGBA{0, 0, 0, 0}, l.outline)
	gocv.PutText(img, text, image.Pt(x, y), gocv.FontHersheyPlain, l.font, clr, l.thickness)
}

// drawRightBlock renders right-aligned lines starting at y and returns the y below the block.
func (l layout) drawRightBlock(img *gocv.Mat, lines []string, y int, clr color.RGBA) int {
	for _, line := range lines {
		x := img.Cols() - l.textWidth(line) - l.pad
		l.drawText(img, line, x, y, clr)
		y += l.line
	}
	return y + l.line/2
}

// menuLines is the content of the control menu; empty strings are spacers.
var menuLines = []string{
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
	"[B] Noise Experiment Start/Stop",
	"[G] Experiment Mode",
	"[H] Experiment Duration",
	"[-/=] Noise Threshold",
	"[V] Noise Graph Window",
	"[U] Mode Matrix Start/Abort",
	"[L] Lock Auto Exposure/WB/Focus",
	"[O] Driver Settings (DirectShow)",
	"",
	"[TAB] Close Menu",
	"[ESC] Camera Menu",
}

// splitMenu returns the menu as one column if it fits in maxLines, otherwise as two columns.
// The split is made at the spacer closest to the middle when both columns still fit,
// otherwise exactly in the middle.
func splitMenu(lines []string, maxLines int) [][]string {
	if len(lines) <= maxLines {
		return [][]string{lines}
	}

	mid := (len(lines) + 1) / 2
	split := -1
	for i, line := range lines {
		fits := i <= maxLines && len(lines)-i-1 <= maxLines
		if line == "" && fits && (split < 0 || abs(i-mid) < abs(split-mid)) {
			split = i
		}
	}
	if split < 0 {
		split = mid
	}

	second := lines[split:]
	if second[0] == "" { // do not start the second column with a spacer
		second = second[1:]
	}
	return [][]string{lines[:split], second}
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

// drawMenu renders the control menu, in two columns when it does not fit the frame height.
func drawMenu(img *gocv.Mat, l layout) {
	available := (img.Rows() - l.top - bottomLines*l.line) / l.line
	columns := splitMenu(menuLines, available)

	x := l.pad
	for _, column := range columns {
		y := l.top
		width := 0
		for _, line := range column {
			if line != "" {
				l.drawText(img, line, x, y, color.RGBA{255, 255, 255, 0})
				width = max(width, l.textWidth(line))
			}
			y += l.line
		}
		x += width + 2*l.pad
	}
}

// drawStats renders performance statistics and returns the y below them.
func drawStats(img *gocv.Mat, l layout, y int, fps float64, latency int64, width, height int, format, cameraName string) int {
	stats := []string{
		fmt.Sprintf("FPS: %.2f", fps),
		fmt.Sprintf("Latency: %d ms", latency),
		fmt.Sprintf("Resolution: %dx%d", width, height),
		fmt.Sprintf("Format: %s", format),
		fmt.Sprintf("Camera: %s", cameraName),
	}
	return l.drawRightBlock(img, stats, y, color.RGBA{255, 165, 0, 0})
}

// drawSettings renders current filter settings and returns the y below them.
func drawSettings(img *gocv.Mat, l layout, y int, params *control.PipelineParams) int {
	params.RLock()
	settings := []string{
		fmt.Sprintf("Brightness: %.0f", params.BrightnessContrast.Beta),
		fmt.Sprintf("Contrast: %.2f", params.BrightnessContrast.Alpha),
		fmt.Sprintf("Blur Kernel: %d", params.Blur.Ksize),
		fmt.Sprintf("Edge T1: %.0f", params.Edge.Threshold1),
		fmt.Sprintf("Edge T2: %.0f", params.Edge.Threshold2),
	}
	params.RUnlock()

	return l.drawRightBlock(img, settings, y, color.RGBA{0, 255, 255, 0})
}

// drawStatuses renders filter enable statuses and returns the y below them.
func drawStatuses(img *gocv.Mat, l layout, y int, stages []pipeline.StageToggle) int {
	lines := make([]string, 0, len(stages))
	for _, st := range stages {
		lines = append(lines, fmt.Sprintf("%s [%s]", st.Name, helpers.OnOff(st.Enabled)))
	}
	return l.drawRightBlock(img, lines, y, color.RGBA{0, 255, 0, 0})
}

// drawControls renders filter control bindings and returns the y below them.
func drawControls(img *gocv.Mat, l layout, y int) int {
	controls := []string{
		"Brightness: W / S",
		"Contrast:   A / D",
		"Blur Size:  E / Q",
		"Edge T1:    J / K",
		"Edge T2:    N / M",
	}
	return l.drawRightBlock(img, controls, y, color.RGBA{200, 200, 200, 0})
}

// ExperimentInfo is what the experiment panel shows.
type ExperimentInfo struct {
	Mode      string
	Duration  string
	Threshold float64
	Running   bool
	Elapsed   string
	Noise     float64
	HasNoise  bool
	Level     float64
	Light     string // modulator light state, "" in other modes
	Matrix    string // mode matrix progress, "" when no matrix runs
}

// drawExperiment renders experiment settings and the live noise level against the threshold
// in the bottom-left corner, above the notification line.
func drawExperiment(img *gocv.Mat, l layout, info ExperimentInfo) {
	x := l.pad
	y := img.Rows() - l.top - 2*l.line

	if info.Matrix != "" {
		l.drawText(img, "MATRIX "+info.Matrix+"  [U] abort", x, y-l.line, color.RGBA{255, 200, 0, 0})
	}

	if info.Running {
		line := fmt.Sprintf("EXP %s  %s / %s  [B] stop", info.Mode, info.Elapsed, info.Duration)
		if info.Light != "" {
			line = fmt.Sprintf("EXP %s [%s]  %s / %s  [B] stop", info.Mode, info.Light, info.Elapsed, info.Duration)
		}
		l.drawText(img, line, x, y, color.RGBA{255, 80, 80, 0})
	} else {
		line := fmt.Sprintf("Exp: %s [G] | %s [H] | thr %.1f DN [-/=]", info.Mode, info.Duration, info.Threshold)
		l.drawText(img, line, x, y, color.RGBA{255, 255, 255, 0})
	}

	y += l.line

	if !info.HasNoise {
		l.drawText(img, fmt.Sprintf("Noise -- | Level %.1f", info.Level), x, y, color.RGBA{200, 200, 200, 0})
		return
	}

	status, clr := "OK", color.RGBA{0, 255, 0, 0}
	if info.Noise > info.Threshold {
		status, clr = "HIGH", color.RGBA{255, 0, 0, 0}
	}

	line := fmt.Sprintf("Noise %.2f / %.1f DN %s | Level %.1f", info.Noise, info.Threshold, status, info.Level)
	l.drawText(img, line, x, y, clr)
}

// drawNotification renders notification text at the bottom of the frame.
func drawNotification(img *gocv.Mat, l layout, text string) {
	l.drawText(img, text, l.pad, img.Rows()-l.top, color.RGBA{255, 0, 0, 0})
}
