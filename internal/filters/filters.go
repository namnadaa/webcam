package filters

import (
	"fmt"
	"image"

	"gocv.io/x/gocv"
)

const (
	defaultKernelSize = 3
	maxKernelSize     = 31

	defaultThresholdLow  = 50
	defaultThresholdHigh = 150

	maxAlpha = 3.0
	maxBeta  = 100.0
)

// Gray converts the input image to grayscale.
func Gray(src gocv.Mat) (gocv.Mat, error) {
	if src.Empty() {
		return gocv.NewMat(), fmt.Errorf("source image is empty")
	}

	dst := gocv.NewMat()
	err := gocv.CvtColor(src, &dst, gocv.ColorBGRToGray)
	if err != nil {
		return gocv.NewMat(), fmt.Errorf("failed to convert to grayscale: %w", err)
	}

	return dst, nil
}

// Blur applies Gaussian blur to the input image.
func Blur(src gocv.Mat, ksize int) (gocv.Mat, error) {
	if src.Empty() {
		return gocv.NewMat(), fmt.Errorf("source image is empty")
	}

	if ksize <= 0 {
		ksize = defaultKernelSize
	}

	if ksize > maxKernelSize {
		ksize = maxKernelSize
	}

	if ksize%2 == 0 {
		ksize++
	}

	dst := gocv.NewMat()
	err := gocv.GaussianBlur(src, &dst, image.Pt(ksize, ksize), 0, 0, gocv.BorderDefault)
	if err != nil {
		return gocv.NewMat(), fmt.Errorf("failed to apply Gaussian blur: %w", err)
	}

	return dst, nil
}

// Edge detects edges using the Canny algorithm.
func Edge(src gocv.Mat, threshold1, threshold2 float32) (gocv.Mat, error) {
	if src.Empty() {
		return gocv.NewMat(), fmt.Errorf("source image is empty")
	}

	if threshold1 <= 0 {
		threshold1 = defaultThresholdLow
	}

	if threshold2 == 0 || threshold2 <= threshold1 {
		threshold2 = defaultThresholdHigh
	}

	dst := gocv.NewMat()
	err := gocv.Canny(src, &dst, threshold1, threshold2)
	if err != nil {
		return gocv.NewMat(), fmt.Errorf("failed to detects edges: %w", err)
	}

	return dst, nil
}

// BrightnessContrast adjusts brightness and contrast of the image.
func BrightnessContrast(src gocv.Mat, alpha, beta float64) (gocv.Mat, error) {
	if src.Empty() {
		return gocv.NewMat(), fmt.Errorf("source image is empty")
	}

	if alpha <= 0 {
		alpha = 1.0
	}

	if alpha > maxAlpha {
		alpha = maxAlpha
	}

	if beta > maxBeta {
		beta = maxBeta
	}

	dst := gocv.NewMat()
	err := gocv.AddWeighted(src, alpha, gocv.NewMat(), beta, 0, &dst)
	if err != nil {
		return gocv.NewMat(), fmt.Errorf("failed to adjust brightness and contrast: %w", err)
	}

	return dst, nil
}

// Sharpen increases image sharpness using a convolution kernel.
func Sharpen(src gocv.Mat) (gocv.Mat, error) {
	if src.Empty() {
		return gocv.NewMat(), fmt.Errorf("source image is empty")
	}

	dst := gocv.NewMat()
	kernel := gocv.NewMatWithSize(3, 3, gocv.MatTypeCV64F)
	defer kernel.Close()

	kernel.SetDoubleAt(0, 1, -1)
	kernel.SetDoubleAt(1, 0, -1)
	kernel.SetDoubleAt(1, 1, 5)
	kernel.SetDoubleAt(1, 2, -1)
	kernel.SetDoubleAt(2, 1, -1)

	err := gocv.Filter2D(src, &dst, -1, kernel, image.Pt(-1, -1), 0, gocv.BorderDefault)
	if err != nil {
		return gocv.NewMat(), fmt.Errorf("failed to sharpen image: %w", err)
	}
	return dst, nil
}
