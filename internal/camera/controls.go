package camera

import (
	"fmt"

	"gocv.io/x/gocv"
)

// Controls holds the readback of camera properties that affect the signal level.
// Backends that do not support a property report 0 for it.
type Controls struct {
	AutoExposure  float64
	Exposure      float64
	Gain          float64
	AutoWB        float64
	WBTemperature float64
	AutoFocus     float64
}

// String formats controls for logs and reports.
func (c Controls) String() string {
	return fmt.Sprintf(
		"auto_exposure=%g exposure=%g gain=%g auto_wb=%g wb_temperature=%g auto_focus=%g",
		c.AutoExposure, c.Exposure, c.Gain, c.AutoWB, c.WBTemperature, c.AutoFocus,
	)
}

// ReadControls reads the current exposure, gain, white balance and focus settings.
func (c *Camera) ReadControls() Controls {
	return Controls{
		AutoExposure:  c.Get(gocv.VideoCaptureAutoExposure),
		Exposure:      c.Get(gocv.VideoCaptureExposure),
		Gain:          c.Get(gocv.VideoCaptureGain),
		AutoWB:        c.Get(gocv.VideoCaptureAutoWB),
		WBTemperature: c.Get(gocv.VideoCaptureWBTemperature),
		AutoFocus:     c.Get(gocv.VideoCaptureAutoFocus),
	}
}

// manualExposureValues are the "manual" values for auto exposure used by different backends:
// 0.25 for DirectShow/MSMF, 1 for V4L2 and 0 for backends that treat the property as a flag.
var manualExposureValues = []float64{0.25, 1, 0}

// LockAuto turns off auto exposure, auto white balance and autofocus so the camera
// does not adjust the signal during a measurement.
// Returns the controls before and after; the caller keeps "before" to restore them later.
func (c *Camera) LockAuto() (before, after Controls) {
	before = c.ReadControls()

	for _, v := range manualExposureValues {
		if before.AutoExposure == v {
			break
		}
		c.Set(gocv.VideoCaptureAutoExposure, v)
		if c.Get(gocv.VideoCaptureAutoExposure) != before.AutoExposure {
			break
		}
	}

	c.Set(gocv.VideoCaptureAutoWB, 0)
	c.Set(gocv.VideoCaptureAutoFocus, 0)

	return before, c.ReadControls()
}

// UnlockAuto restores the automatic controls saved by LockAuto.
func (c *Camera) UnlockAuto(saved Controls) {
	c.Set(gocv.VideoCaptureAutoExposure, saved.AutoExposure)
	c.Set(gocv.VideoCaptureAutoWB, saved.AutoWB)
	c.Set(gocv.VideoCaptureAutoFocus, saved.AutoFocus)
}

// LockApplied reports which automatic controls changed between two readbacks.
// An empty result means the backend ignored the request.
func LockApplied(before, after Controls) []string {
	var changed []string
	if before.AutoExposure != after.AutoExposure {
		changed = append(changed, "exposure")
	}
	if before.AutoWB != after.AutoWB {
		changed = append(changed, "white balance")
	}
	if before.AutoFocus != after.AutoFocus {
		changed = append(changed, "focus")
	}
	return changed
}
