package control

// BlurParams — Gaussian Blur parameters.
type BlurParams struct {
	Ksize int // the size of the blur core (must be odd: 3, 5, 7...)
}

// EdgeParams — parameters of the Canny boundary detector.
type EdgeParams struct {
	Threshold1 float32 // lower sensitivity threshold
	Threshold2 float32 // upper sensitivity threshold
}

// BrightnessContrastParams — brightness and contrast parameters.
type BrightnessContrastParams struct {
	Alpha float64 // contrast ratio
	Beta  float64 // brightness shift
}

// PipelineParams is a common set of parameters for all pipeline stages.
type PipelineParams struct {
	Blur       BlurParams
	Edge       EdgeParams
	Brightness BrightnessContrastParams
}
