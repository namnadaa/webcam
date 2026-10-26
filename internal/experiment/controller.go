package experiment

import (
	"log/slog"
	"time"
	"webcam/internal/i18n"

	"gocv.io/x/gocv"
)

// Controller keeps the selected experiment settings, measures every frame
// and drives the running session.
type Controller struct {
	Config  Config
	meter   *Meter
	session *Session
	last    Sample
}

// NewController creates a controller with default settings.
func NewController() *Controller {
	return &Controller{
		Config: DefaultConfig(),
		meter:  NewMeter(),
	}
}

// Close stops a running experiment and releases resources.
func (c *Controller) Close() {
	if c.session != nil {
		c.Stop()
	}
	c.meter.Close()
}

// Running reports whether an experiment is in progress.
func (c *Controller) Running() bool {
	return c.session != nil
}

// Last returns the measurements of the most recent frame.
func (c *Controller) Last() Sample {
	return c.last
}

// Elapsed returns the running time of the current experiment.
func (c *Controller) Elapsed() time.Duration {
	if c.session == nil {
		return 0
	}
	return c.session.Elapsed()
}

// ResetMeter forgets the previous frame, call it after the camera was reopened.
func (c *Controller) ResetMeter() {
	c.meter.Reset()
}

// NextMode switches the experiment mode. Settings are fixed while an experiment runs.
func (c *Controller) NextMode() string {
	if c.Running() {
		return i18n.T("Stop the experiment first ([B])")
	}
	c.Config.Mode = c.Config.Mode.Next()
	return i18n.Tf("Mode: %s - %s", c.Config.Mode.Title(), c.Config.Mode.Hint())
}

// NextDuration switches the experiment duration.
func (c *Controller) NextDuration() string {
	if c.Running() {
		return i18n.T("Stop the experiment first ([B])")
	}
	c.Config.NextDuration()
	return i18n.Tf("Duration: %s", FormatDuration(c.Config.Duration()))
}

// AdjustThreshold changes the noise threshold by delta.
func (c *Controller) AdjustThreshold(delta float64) string {
	if c.Running() {
		return i18n.T("Stop the experiment first ([B])")
	}
	c.Config.AdjustThreshold(delta)
	return i18n.Tf("Noise threshold: %.1f DN", c.Config.Threshold)
}

// Toggle starts a new experiment or stops the running one.
func (c *Controller) Toggle(meta Meta) string {
	if c.Running() {
		return c.Stop()
	}

	s, err := Start(c.Config, meta, meta.Height, meta.Width)
	if err != nil {
		slog.Error("experiment start", "err", err)
		return i18n.Tf("Experiment not started: %v", err)
	}

	c.session = s
	return i18n.Tf("Experiment %s %s started: %s", c.Config.Mode.Title(), FormatDuration(c.Config.Duration()), c.Config.Mode.Hint())
}

// Stop ends the running experiment early and saves the results.
func (c *Controller) Stop() string {
	msg, _ := c.finish(true)
	return msg
}

// LightState returns the light state of the last frame in the modulator mode, or "".
func (c *Controller) LightState() string {
	if c.session == nil || c.session.Config().Mode != ModeModulator {
		return ""
	}
	return c.session.State().Title()
}

// Update measures a raw camera frame and feeds it to the running experiment.
// When the experiment has just finished on its own, returns a message and its summary.
func (c *Controller) Update(raw gocv.Mat, t time.Time) (string, *Summary) {
	c.last = c.meter.Measure(raw, t)

	if c.session != nil && c.session.Add(c.last, c.meter.Gray()) {
		return c.finish(false)
	}
	return "", nil
}

// finish saves the session results and returns a message for the user and the summary.
// The summary is returned even on a save error, so a mode matrix can still record the step.
func (c *Controller) finish(stopped bool) (string, *Summary) {
	summary, err := c.session.Finish(stopped)
	c.session = nil

	if err != nil {
		slog.Error("experiment finish", "err", err)
		return i18n.Tf("Experiment save error: %v", err), &summary
	}
	return i18n.Tf("Experiment done: %s", summary.Text()), &summary
}
