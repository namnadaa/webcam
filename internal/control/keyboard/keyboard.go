package keyboard

import (
	"webcam/internal/actions"
	"webcam/internal/control"
	"webcam/internal/media"
	"webcam/internal/pipeline"
	"webcam/internal/ui"

	"gocv.io/x/gocv"
)

// HandleKeyboard — changes parameters by pressing buttons.
// Returns false if the program needs to be terminated.
func HandleKeyboard(win *gocv.Window, params *control.PipelineParams, stages []pipeline.StageToggle, uiState *ui.State, mediaState *media.State) (bool, bool, actions.CameraAction) {
	key := win.WaitKey(1)
	stageChanged := false
	action := actions.ActionNone

	switch key {
	case 27: // ESC
		return false, false, actions.ActionNone

	case 9: // TAB
		uiState.ShowMenu = !uiState.ShowMenu

	case keyArrowLeft: // LEFT
		action = actions.ActionPrevResolution

	case keyArrowRight: // RIGHT
		action = actions.ActionNextResolution

	case keyArrowUp: // UP
		action = actions.ActionNextFPS

	case keyArrowDown: // DOWN
		action = actions.ActionPrevFPS

	case int('t'), int('T'): // statistics
		uiState.ShowStats = !uiState.ShowStats

	case int('p'), int('P'): // settings
		uiState.ShowSettings = !uiState.ShowSettings

	case int('f'), int('F'): // statuses
		uiState.ShowStatuses = !uiState.ShowStatuses

	case int('c'), int('C'): // controls
		uiState.ShowControls = !uiState.ShowControls

	case int('x'), int('X'): // screenshot
		mediaState.Screenshot = true

	case int('r'), int('R'): // recording video
		mediaState.Recording = !mediaState.Recording
		stageChanged = true

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
		params.Lock()
		params.BrightnessContrast.Beta += 5
		params.Unlock()
	case int('s'), int('S'):
		params.Lock()
		params.BrightnessContrast.Beta -= 5
		params.Unlock()

		// ===== Contrast =====
	case int('d'), int('D'):
		params.Lock()
		params.BrightnessContrast.Alpha += 0.1
		params.Unlock()
	case int('a'), int('A'):
		params.Lock()
		params.BrightnessContrast.Alpha -= 0.1
		params.Unlock()

		// ===== Blur size (Gaussian) =====
	case int('e'), int('E'):
		params.Lock()
		params.Blur.Ksize += 2
		if params.Blur.Ksize%2 == 0 {
			params.Blur.Ksize++
		}
		params.Unlock()
	case int('q'), int('Q'):
		params.Lock()
		if params.Blur.Ksize > 3 {
			params.Blur.Ksize -= 2
			if params.Blur.Ksize%2 == 0 {
				params.Blur.Ksize--
			}
		}
		params.Unlock()

		// ===== Canny thresholds =====
	case int('j'), int('J'):
		params.Lock()
		params.Edge.Threshold1 -= 5
		params.Unlock()
	case int('k'), int('K'):
		params.Lock()
		params.Edge.Threshold1 += 5
		params.Unlock()
	case int('n'), int('N'):
		params.Lock()
		params.Edge.Threshold2 -= 5
		params.Unlock()
	case int('m'), int('M'):
		params.Lock()
		params.Edge.Threshold2 += 5
		params.Unlock()
	}

	return true, stageChanged, action
}
