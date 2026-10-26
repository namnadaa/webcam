package camera

import (
	"fmt"
	"webcam/internal/i18n"

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
	Focus         float64
}

// Reported reports whether the backend returned any of the controls.
// AVFoundation (macOS) returns 0 for all of them.
func (c Controls) Reported() bool {
	return c != Controls{}
}

// String formats controls for logs and reports.
func (c Controls) String() string {
	return fmt.Sprintf(
		"auto_exposure=%g exposure=%g gain=%g auto_wb=%g wb_temperature=%g auto_focus=%g focus=%g",
		c.AutoExposure, c.Exposure, c.Gain, c.AutoWB, c.WBTemperature, c.AutoFocus, c.Focus,
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
		Focus:         c.Get(gocv.VideoCaptureFocus),
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

	// DirectShow and Media Foundation switch a control to manual when its value is written,
	// so writing back the current values freezes them even when the auto flags are ignored
	for _, w := range []struct {
		prop gocv.VideoCaptureProperties
		val  float64
	}{
		{gocv.VideoCaptureExposure, before.Exposure},
		{gocv.VideoCaptureWBTemperature, before.WBTemperature},
		{gocv.VideoCaptureFocus, before.Focus},
	} {
		if w.val != 0 {
			c.Set(w.prop, w.val)
		}
	}

	return before, c.ReadControls()
}

// UnlockAuto restores the automatic controls saved by LockAuto.
func (c *Camera) UnlockAuto(saved Controls) {
	c.Set(gocv.VideoCaptureAutoExposure, saved.AutoExposure)
	c.Set(gocv.VideoCaptureAutoWB, saved.AutoWB)
	c.Set(gocv.VideoCaptureAutoFocus, saved.AutoFocus)
}

// LockApplied reports which automatic controls were switched off according to the readback.
func LockApplied(before, after Controls) []string {
	var changed []string
	if before.AutoExposure != after.AutoExposure {
		changed = append(changed, i18n.T("exposure"))
	}
	if before.AutoWB != after.AutoWB {
		changed = append(changed, i18n.T("white balance"))
	}
	if before.AutoFocus != after.AutoFocus {
		changed = append(changed, i18n.T("focus"))
	}
	return changed
}

// ValuesFixed reports which control values LockAuto wrote back to switch them to manual.
// The backend may not report the auto flags, so this is the only trace of the lock then.
func ValuesFixed(before Controls) []string {
	var fixed []string
	if before.Exposure != 0 {
		fixed = append(fixed, i18n.T("exposure"))
	}
	if before.WBTemperature != 0 {
		fixed = append(fixed, i18n.T("white balance"))
	}
	if before.Focus != 0 {
		fixed = append(fixed, i18n.T("focus"))
	}
	return fixed
}
