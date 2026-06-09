package filters_test

import (
	"image"
	"image/color"
	"testing"
	"webcam/internal/filters"

	"gocv.io/x/gocv"
)

func createTestImage() gocv.Mat {
	img := gocv.NewMatWithSize(100, 100, gocv.MatTypeCV8UC3)
	rect := image.Rect(20, 20, 80, 80)
	gocv.Rectangle(&img, rect, colorRGB(0, 0, 255), -1)
	return img
}

func colorRGB(r, g, b uint8) color.RGBA {
	return color.RGBA{R: r, G: g, B: b, A: 0}
}

func TestFilters(t *testing.T) {
	tests := []struct {
		name    string
		run     func(src gocv.Mat) (gocv.Mat, error)
		wantErr bool
	}{
		{
			name: "Gray conversion",
			run: func(src gocv.Mat) (gocv.Mat, error) {
				return filters.Gray(src)
			},
		},
		{
			name: "Gaussian Blur with ksize=5",
			run: func(src gocv.Mat) (gocv.Mat, error) {
				return filters.Blur(src, 5)
			},
		},
		{
			name: "Gaussian Blur with invalid ksize (0)",
			run: func(src gocv.Mat) (gocv.Mat, error) {
				return filters.Blur(src, 0)
			},
		},
		{
			name: "Edge detection (valid thresholds)",
			run: func(src gocv.Mat) (gocv.Mat, error) {
				return filters.Edge(src, 50, 150)
			},
		},
		{
			name: "Edge detection (invalid thresholds)",
			run: func(src gocv.Mat) (gocv.Mat, error) {
				return filters.Edge(src, 0, 0)
			},
		},
		{
			name: "Brightness and contrast adjustment (valid)",
			run: func(src gocv.Mat) (gocv.Mat, error) {
				return filters.BrightnessContrast(src, 1.5, 20)
			},
		},
		{
			name: "Brightness and contrast adjustment (out of range alpha/beta)",
			run: func(src gocv.Mat) (gocv.Mat, error) {
				return filters.BrightnessContrast(src, 1000, 500)
			},
		},
		{
			name: "Brightness and contrast adjustment (empty source)",
			run: func(_ gocv.Mat) (gocv.Mat, error) {
				empty := gocv.NewMat()
				defer empty.Close()
				return filters.BrightnessContrast(empty, 1.0, 0)
			},
			wantErr: true,
		},
		{
			name: "Sharpen filter",
			run: func(src gocv.Mat) (gocv.Mat, error) {
				return filters.Sharpen(src)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var src gocv.Mat
			if !tt.wantErr {
				src = createTestImage()
				defer src.Close()
			}

			dst, err := tt.run(src)

			if tt.wantErr {
				dst.Close()
				if err == nil {
					t.Fatalf("%s: expected error but got nil", tt.name)
				}
				return
			}

			defer dst.Close()

			if err != nil {
				t.Fatalf("%s: unexpected error: %v", tt.name, err)
			}

			if dst.Empty() {
				t.Errorf("%s: result is empty", tt.name)
			}

			if dst.Rows() != src.Rows() || dst.Cols() != src.Cols() {
				t.Errorf("%s: size mismatch (got %dx%d, want %dx%d)",
					tt.name, dst.Rows(), dst.Cols(), src.Rows(), src.Cols())
			}
		})
	}
}
