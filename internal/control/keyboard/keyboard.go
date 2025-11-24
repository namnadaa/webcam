package keyboard

import (
	"webcam/internal/control"
	"webcam/internal/pipeline"

	"gocv.io/x/gocv"
)

// HandleKeyboard — changes parameters by pressing buttons.
// Returns false if the program needs to be terminated.
func HandleKeyboard(win *gocv.Window, params *control.PipelineParams, stages []pipeline.StageToggle) (bool, bool) {
	key := win.WaitKey(1)
	stageChanged := false

	switch key {
	case 27: // ESC
		return false, false

	case 49: // brightness/contrast
		stages[0].Enabled = !stages[0].Enabled
		stageChanged = true
	case 50: // blur
		stages[1].Enabled = !stages[1].Enabled
		stageChanged = true
	case 51: // edge
		stages[2].Enabled = !stages[2].Enabled
		stageChanged = true
	case 52: // gray
		stages[3].Enabled = !stages[3].Enabled
		stageChanged = true
	case 53: // sharpen
		stages[4].Enabled = !stages[4].Enabled
		stageChanged = true

		// ===== Brightness =====
	case int('w'), int('W'):
		params.BrightnessContrast.Beta += 5
	case int('s'), int('S'):
		params.BrightnessContrast.Beta -= 5

		// ===== Contrast =====
	case int('d'), int('D'):
		params.BrightnessContrast.Alpha += 0.1
	case int('a'), int('A'):
		params.BrightnessContrast.Alpha -= 0.1

		// ===== Blur size (Gaussian) =====
	case int('e'), int('E'):
		params.Blur.Ksize += 2
		if params.Blur.Ksize%2 == 0 {
			params.Blur.Ksize++
		}
	case int('q'), int('Q'):
		if params.Blur.Ksize > 3 {
			params.Blur.Ksize -= 2
			if params.Blur.Ksize%2 == 0 {
				params.Blur.Ksize--
			}
		}

		// ===== Canny thresholds =====
	case int('j'), int('J'):
		params.Edge.Threshold1 -= 5
	case int('k'), int('K'):
		params.Edge.Threshold1 += 5
	case int('n'), int('N'):
		params.Edge.Threshold2 -= 5
	case int('m'), int('M'):
		params.Edge.Threshold2 += 5
	}

	return true, stageChanged
}
