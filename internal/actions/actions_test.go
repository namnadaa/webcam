package actions_test

import (
	"testing"
	"webcam/internal/actions"
	"webcam/internal/camera"
)

func TestHandleAction_None(t *testing.T) {
	cfg := camera.Config{Width: 640, Height: 480, FPS: 30}
	resIdx, fpsIdx := 1, 2

	changed := actions.HandleAction(actions.ActionNone, &cfg, &resIdx, &fpsIdx)

	if changed {
		t.Error("ActionNone: expected cameraChanged=false")
	}
	if resIdx != 1 || fpsIdx != 2 {
		t.Errorf("ActionNone: indices changed: resIdx=%d fpsIdx=%d", resIdx, fpsIdx)
	}
	if cfg.Width != 640 || cfg.Height != 480 || cfg.FPS != 30 {
		t.Errorf("ActionNone: cfg changed: %+v", cfg)
	}
}

func TestHandleAction_NextResolution(t *testing.T) {
	tests := []struct {
		name        string
		startIdx    int
		wantIdx     int
		wantChanged bool
	}{
		{"normal advance", 0, 1, true},
		{"middle", 4, 5, true},
		{"wrap around at last", len(camera.Resolutions) - 1, 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := camera.Config{}
			resIdx := tt.startIdx
			fpsIdx := 0

			changed := actions.HandleAction(actions.ActionNextResolution, &cfg, &resIdx, &fpsIdx)

			if changed != tt.wantChanged {
				t.Errorf("cameraChanged = %v, want %v", changed, tt.wantChanged)
			}
			if resIdx != tt.wantIdx {
				t.Errorf("resolutionIndex = %d, want %d", resIdx, tt.wantIdx)
			}
			want := camera.Resolutions[tt.wantIdx]
			if cfg.Width != want.Width || cfg.Height != want.Height {
				t.Errorf("cfg resolution = %dx%d, want %dx%d", cfg.Width, cfg.Height, want.Width, want.Height)
			}
		})
	}
}

func TestHandleAction_PrevResolution(t *testing.T) {
	tests := []struct {
		name     string
		startIdx int
		wantIdx  int
	}{
		{"normal step back", 3, 2},
		{"from first wraps to last", 0, len(camera.Resolutions) - 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := camera.Config{}
			resIdx := tt.startIdx
			fpsIdx := 0

			changed := actions.HandleAction(actions.ActionPrevResolution, &cfg, &resIdx, &fpsIdx)

			if !changed {
				t.Error("cameraChanged = false, want true")
			}
			if resIdx != tt.wantIdx {
				t.Errorf("resolutionIndex = %d, want %d", resIdx, tt.wantIdx)
			}
			want := camera.Resolutions[tt.wantIdx]
			if cfg.Width != want.Width || cfg.Height != want.Height {
				t.Errorf("cfg resolution = %dx%d, want %dx%d", cfg.Width, cfg.Height, want.Width, want.Height)
			}
		})
	}
}

func TestHandleAction_NextFPS(t *testing.T) {
	tests := []struct {
		name     string
		startIdx int
		wantIdx  int
	}{
		{"normal advance", 0, 1},
		{"wrap around at last", len(camera.FPSPresets) - 1, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := camera.Config{}
			resIdx := 0
			fpsIdx := tt.startIdx

			changed := actions.HandleAction(actions.ActionNextFPS, &cfg, &resIdx, &fpsIdx)

			if !changed {
				t.Error("cameraChanged = false, want true")
			}
			if fpsIdx != tt.wantIdx {
				t.Errorf("fpsIndex = %d, want %d", fpsIdx, tt.wantIdx)
			}
			if cfg.FPS != camera.FPSPresets[tt.wantIdx] {
				t.Errorf("cfg.FPS = %v, want %v", cfg.FPS, camera.FPSPresets[tt.wantIdx])
			}
		})
	}
}

func TestHandleAction_PrevFPS(t *testing.T) {
	tests := []struct {
		name     string
		startIdx int
		wantIdx  int
	}{
		{"normal step back", 3, 2},
		{"from first wraps to last", 0, len(camera.FPSPresets) - 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := camera.Config{}
			resIdx := 0
			fpsIdx := tt.startIdx

			changed := actions.HandleAction(actions.ActionPrevFPS, &cfg, &resIdx, &fpsIdx)

			if !changed {
				t.Error("cameraChanged = false, want true")
			}
			if fpsIdx != tt.wantIdx {
				t.Errorf("fpsIndex = %d, want %d", fpsIdx, tt.wantIdx)
			}
			if cfg.FPS != camera.FPSPresets[tt.wantIdx] {
				t.Errorf("cfg.FPS = %v, want %v", cfg.FPS, camera.FPSPresets[tt.wantIdx])
			}
		})
	}
}
