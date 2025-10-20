package filters

import (
	"image"

	"gocv.io/x/gocv"
)

const (
	defaultKernelSize    = 3
	maxKernelSize        = 31
	defaultThresholdLow  = 50
	defaultThresholdHigh = 150
)

// Gray converts the input image to grayscale.
func Gray(src gocv.Mat) gocv.Mat {
	dst := gocv.NewMat()
	gocv.CvtColor(src, &dst, gocv.ColorBGRToGray)
	return dst
}

// Blur applies Gaussian blur to the input image.
func Blur(src gocv.Mat, ksize int) gocv.Mat {
	if ksize <= 0 {
		ksize = defaultKernelSize
	}

	if ksize > 31 {
		ksize = maxKernelSize
	}

	if ksize%2 == 0 {
		ksize++
	}

	dst := gocv.NewMat()
	gocv.GaussianBlur(src, &dst, image.Pt(ksize, ksize), 0, 0, gocv.BorderDefault)
	return dst
}

// Edge detects edges using the Canny algorithm.
func Edge(src gocv.Mat, threshold1, threshold2 float32) gocv.Mat {
	if threshold1 <= 0 {
		threshold1 = defaultThresholdLow
	}

	if threshold2 == 0 || threshold2 <= threshold1 {
		threshold2 = defaultThresholdHigh
	}

	dst := gocv.NewMat()
	gocv.Canny(src, &dst, threshold1, threshold2)
	return dst
}
