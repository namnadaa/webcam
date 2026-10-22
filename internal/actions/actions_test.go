package actions_test

import (
	"testing"
	"webcam/internal/actions"
	"webcam/internal/camera"
)

var testModes = []camera.Mode{
	{Width: 320, Height: 240, FPS: []float64{15, 30}},
	{Width: 640, Height: 480, FPS: []float64{15, 30, 60}},
	{Width: 1280, Height: 720, FPS: []float64{10}},
	{Width: 1920, Height: 1080, FPS: []float64{5, 30}},
}

func TestHandleAction_None(t *testing.T) {
	cfg := camera.Config{Width: 640, Height: 480, FPS: 30}
	resIdx, fpsIdx := 1, 1

	changed := actions.HandleAction(actions.ActionNone, testModes, &cfg, &resIdx, &fpsIdx)

	if changed {
		t.Error("ActionNone: expected cameraChanged=false")
	}
	if resIdx != 1 || fpsIdx != 1 {
		t.Errorf("ActionNone: indices changed: resIdx=%d fpsIdx=%d", resIdx, fpsIdx)
	}
	if cfg.Width != 640 || cfg.Height != 480 || cfg.FPS != 30 {
		t.Errorf("ActionNone: cfg changed: %+v", cfg)
	}
}

func TestHandleAction_EmptyModes(t *testing.T) {
	cfg := camera.Config{Width: 640, Height: 480, FPS: 30}
	resIdx, fpsIdx := 0, 0

	if actions.HandleAction(actions.ActionNextResolution, nil, &cfg, &resIdx, &fpsIdx) {
		t.Error("expected cameraChanged=false without modes")
	}
}

func TestHandleAction_NextResolution(t *testing.T) {
	tests := []struct {
		name        string
		startIdx    int
		startFPS    float64
		wantIdx     int
		wantFPS     float64
		wantChanged bool
	}{
		{"normal advance keeps fps", 0, 30, 1, 30, true},
		{"fps snaps to closest", 1, 60, 2, 10, true},
		{"wrap around at last", len(testModes) - 1, 30, 0, 30, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := camera.Config{FPS: tt.startFPS}
			resIdx := tt.startIdx
			fpsIdx := 0

			changed := actions.HandleAction(actions.ActionNextResolution, testModes, &cfg, &resIdx, &fpsIdx)

			if changed != tt.wantChanged {
				t.Errorf("cameraChanged = %v, want %v", changed, tt.wantChanged)
			}
			if resIdx != tt.wantIdx {
				t.Errorf("resolutionIndex = %d, want %d", resIdx, tt.wantIdx)
			}
			want := testModes[tt.wantIdx]
			if cfg.Width != want.Width || cfg.Height != want.Height {
				t.Errorf("cfg resolution = %dx%d, want %dx%d", cfg.Width, cfg.Height, want.Width, want.Height)
			}
			if cfg.FPS != tt.wantFPS {
				t.Errorf("cfg.FPS = %v, want %v", cfg.FPS, tt.wantFPS)
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
		{"from first wraps to last", 0, len(testModes) - 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := camera.Config{}
			resIdx := tt.startIdx
			fpsIdx := 0

			changed := actions.HandleAction(actions.ActionPrevResolution, testModes, &cfg, &resIdx, &fpsIdx)

			if !changed {
				t.Error("cameraChanged = false, want true")
			}
			if resIdx != tt.wantIdx {
				t.Errorf("resolutionIndex = %d, want %d", resIdx, tt.wantIdx)
			}
			want := testModes[tt.wantIdx]
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
		{"wrap around at last", 2, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := camera.Config{}
			resIdx := 1
			fpsIdx := tt.startIdx

			changed := actions.HandleAction(actions.ActionNextFPS, testModes, &cfg, &resIdx, &fpsIdx)

			if !changed {
				t.Error("cameraChanged = false, want true")
			}
			if fpsIdx != tt.wantIdx {
				t.Errorf("fpsIndex = %d, want %d", fpsIdx, tt.wantIdx)
			}
			if cfg.FPS != testModes[1].FPS[tt.wantIdx] {
				t.Errorf("cfg.FPS = %v, want %v", cfg.FPS, testModes[1].FPS[tt.wantIdx])
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
		{"normal step back", 2, 1},
		{"from first wraps to last", 0, 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := camera.Config{}
			resIdx := 1
			fpsIdx := tt.startIdx

			changed := actions.HandleAction(actions.ActionPrevFPS, testModes, &cfg, &resIdx, &fpsIdx)

			if !changed {
				t.Error("cameraChanged = false, want true")
			}
			if fpsIdx != tt.wantIdx {
				t.Errorf("fpsIndex = %d, want %d", fpsIdx, tt.wantIdx)
			}
			if cfg.FPS != testModes[1].FPS[tt.wantIdx] {
				t.Errorf("cfg.FPS = %v, want %v", cfg.FPS, testModes[1].FPS[tt.wantIdx])
			}
		})
	}
}
