package camera_test

import (
	"testing"
	"webcam/internal/camera"

	"gocv.io/x/gocv"
)

func openTestCamera(t *testing.T, cfg camera.Config) *camera.Camera {
	t.Helper()
	cam, err := camera.Open(cfg)
	if err != nil {
		t.Fatalf("failed to open camera: %v", err)
	}
	return cam
}

func TestOpen(t *testing.T) {
	tests := []struct {
		name    string
		cfg     camera.Config
		wantErr bool
	}{
		{
			"valid index",
			camera.Config{
				Index:  0,
				API:    camera.APIAVFoundation,
				Width:  640,
				Height: 480,
				FPS:    30,
			},
			false,
		},
		{
			"invalid index",
			camera.Config{
				Index: 99,
				API:   camera.APIAVFoundation,
			},
			true,
		},
		{
			"zero API",
			camera.Config{
				Index:  0,
				API:    0,
				Width:  320,
				Height: 240,
			},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, gotErr := camera.Open(tt.cfg)
			if gotErr != nil {
				if !tt.wantErr {
					t.Errorf("Unexpected error: got %v, wantErr=%v", gotErr, tt.wantErr)
				}
				return
			}

			if tt.wantErr {
				t.Fatal("Open() succeeded unexpectedly")
			}

			if got != nil {
				err := got.Close()
				if err != nil {
					t.Errorf("failed to close camera: %v", err)
				}
			}
		})
	}
}

func TestCamera_Read(t *testing.T) {
	tests := []struct {
		name    string
		cfg     camera.Config
		wantErr bool
		want    bool
	}{
		{
			name: "valid camera read",
			cfg: camera.Config{
				Index:  0,
				API:    camera.APIAVFoundation,
				Width:  640,
				Height: 480,
				FPS:    30,
			},
			wantErr: false,
			want:    true,
		},
		{
			name: "invalid camera index",
			cfg: camera.Config{
				Index: 99,
				API:   camera.APIAVFoundation,
			},
			wantErr: true,
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cam, err := camera.Open(tt.cfg)
			if err != nil {
				if !tt.wantErr {
					t.Fatalf("Unexpected error: %v", err)
				}
				return
			}
			defer cam.Close()

			frame := gocv.NewMat()
			defer frame.Close()

			got := cam.Read(&frame)
			if got != tt.want {
				t.Errorf("Read(): got  = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCamera_Properties(t *testing.T) {
	cam := openTestCamera(t, camera.Config{
		Index:  0,
		API:    camera.APIAVFoundation,
		Width:  640,
		Height: 480,
		FPS:    30,
	})
	defer cam.Close()

	tests := []struct {
		name string
		test func(t *testing.T)
	}{
		{
			"Get exposure",
			func(t *testing.T) {
				val := cam.Get(gocv.VideoCaptureExposure)
				t.Logf("Exposure: %v", val)
			},
		},
		{
			"Set width/height",
			func(t *testing.T) {
				cam.Set(gocv.VideoCaptureFrameWidth, 800)
				cam.Set(gocv.VideoCaptureFrameHeight, 600)
				w, h := cam.ActualSize()
				if w == 0 || h == 0 {
					t.Errorf("invalid size: %dx%d", w, h)
				}
			},
		},
		{
			"Check FPS",
			func(t *testing.T) {
				fps := cam.ActualFPS()
				if fps <= 0 {
					t.Errorf("invalid FPS: %v", fps)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, tt.test)
	}
}
