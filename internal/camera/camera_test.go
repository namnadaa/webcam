package camera_test

import (
	"testing"
	"webcam/internal/camera"

	"gocv.io/x/gocv"
)

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
