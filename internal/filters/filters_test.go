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
		name string
		run  func(src gocv.Mat) gocv.Mat
	}{
		{
			name: "Gray conversion",
			run: func(src gocv.Mat) gocv.Mat {
				return filters.Gray(src)
			},
		},
		{
			name: "Gaussian Blur with ksize=5",
			run: func(src gocv.Mat) gocv.Mat {
				return filters.Blur(src, 5)
			},
		},
		{
			name: "Gaussian Blur with invalid ksize (0)",
			run: func(src gocv.Mat) gocv.Mat {
				return filters.Blur(src, 0)
			},
		},
		{
			name: "Edge detection (valid thresholds)",
			run: func(src gocv.Mat) gocv.Mat {
				return filters.Edge(src, 50, 150)
			},
		},
		{
			name: "Edge detection (invalid thresholds)",
			run: func(src gocv.Mat) gocv.Mat {
				return filters.Edge(src, 0, 0)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			src := createTestImage()
			defer src.Close()

			dst := tt.run(src)
			defer dst.Close()

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
