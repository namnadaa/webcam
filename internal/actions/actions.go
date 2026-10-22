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
	ActionToggleAutoLock
	ActionToggleExperiment
	ActionNextExperimentMode
	ActionNextExperimentDuration
	ActionThresholdUp
	ActionThresholdDown
)

// HandleAction updates camera config using keyboard actions.
// Resolutions and frame rates are cycled through the given supported modes.
func HandleAction(action CameraAction, modes []camera.Mode, cfg *camera.Config, resolutionIndex *int, fpsIndex *int) bool {
	if len(modes) == 0 {
		return false
	}

	cameraChanged := false

	switch action {
	case ActionNextResolution:
		*resolutionIndex = wrap(*resolutionIndex+1, len(modes))
		*fpsIndex = camera.ClosestFPS(modes[*resolutionIndex].FPS, cfg.FPS)
		cameraChanged = true

	case ActionPrevResolution:
		*resolutionIndex = wrap(*resolutionIndex-1, len(modes))
		*fpsIndex = camera.ClosestFPS(modes[*resolutionIndex].FPS, cfg.FPS)
		cameraChanged = true

	case ActionNextFPS:
		if n := len(modes[*resolutionIndex].FPS); n > 0 {
			*fpsIndex = wrap(*fpsIndex+1, n)
			cameraChanged = true
		}

	case ActionPrevFPS:
		if n := len(modes[*resolutionIndex].FPS); n > 0 {
			*fpsIndex = wrap(*fpsIndex-1, n)
			cameraChanged = true
		}
	}

	if cameraChanged {
		mode := modes[*resolutionIndex]

		cfg.Width = mode.Width
		cfg.Height = mode.Height
		if len(mode.FPS) > 0 {
			cfg.FPS = mode.FPS[*fpsIndex]
		}
	}

	return cameraChanged
}

// wrap keeps an index inside [0, n) cycling at both ends.
func wrap(i, n int) int {
	return ((i % n) + n) % n
}
