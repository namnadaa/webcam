package actions

import (
	"webcam/internal/camera"
)

// CameraAction represents keyboard actions.
type CameraAction int

const (
	ActionNone CameraAction = iota
	ActionNextResolution
	ActionPrevResolution
	ActionNextFPS
	ActionPrevFPS
)

// HandleAction updates camera config using keyboard actions.
func HandleAction(action CameraAction, cfg *camera.Config, resolutionIndex *int, fpsIndex *int) bool {
	cameraChanged := false

	switch action {
	case ActionNextResolution:
		*(resolutionIndex)++

		if *resolutionIndex >= len(camera.Resolutions) {
			*resolutionIndex = 0
		}

		cameraChanged = true

	case ActionPrevResolution:
		*(resolutionIndex)--

		if *resolutionIndex < 0 {
			*resolutionIndex = len(camera.Resolutions) - 1
		}

		cameraChanged = true

	case ActionNextFPS:
		*(fpsIndex)++

		if *fpsIndex >= len(camera.FPSPresets) {
			*fpsIndex = 0
		}

		cameraChanged = true

	case ActionPrevFPS:
		*(fpsIndex)--

		if *fpsIndex < 0 {
			*fpsIndex = len(camera.FPSPresets) - 1
		}

		cameraChanged = true
	}

	if cameraChanged {
		resolution := camera.Resolutions[*resolutionIndex]

		cfg.Width = resolution.Width
		cfg.Height = resolution.Height
		cfg.FPS = camera.FPSPresets[*fpsIndex]
	}

	return cameraChanged
}
