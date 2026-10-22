package experiment

import (
	"fmt"
	"log/slog"
	"time"

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
		return "Stop the experiment first ([B])"
	}
	c.Config.Mode = c.Config.Mode.Next()
	return fmt.Sprintf("Mode: %s - %s", c.Config.Mode, c.Config.Mode.Hint())
}

// NextDuration switches the experiment duration.
func (c *Controller) NextDuration() string {
	if c.Running() {
		return "Stop the experiment first ([B])"
	}
	c.Config.NextDuration()
	return "Duration: " + FormatDuration(c.Config.Duration())
}

// AdjustThreshold changes the noise threshold by delta.
func (c *Controller) AdjustThreshold(delta float64) string {
	if c.Running() {
		return "Stop the experiment first ([B])"
	}
	c.Config.AdjustThreshold(delta)
	return fmt.Sprintf("Noise threshold: %.1f DN", c.Config.Threshold)
}

// Toggle starts a new experiment or stops the running one.
func (c *Controller) Toggle(meta Meta) string {
	if c.Running() {
		return c.Stop()
	}

	s, err := Start(c.Config, meta, meta.Height, meta.Width)
	if err != nil {
		slog.Error("experiment start", "err", err)
		return "Experiment not started: " + err.Error()
	}

	c.session = s
	return fmt.Sprintf("Experiment %s %s started: %s", c.Config.Mode, FormatDuration(c.Config.Duration()), c.Config.Mode.Hint())
}

// Stop ends the running experiment early and saves the results.
func (c *Controller) Stop() string {
	return c.finish(true)
}

// Update measures a raw camera frame and feeds it to the running experiment.
// Returns a message when the experiment has just finished on its own.
func (c *Controller) Update(raw gocv.Mat, t time.Time) string {
	c.last = c.meter.Measure(raw, t)

	if c.session != nil && c.session.Add(c.last, c.meter.Gray()) {
		return c.finish(false)
	}
	return ""
}

// finish saves the session results and returns a message for the user.
func (c *Controller) finish(stopped bool) string {
	summary, err := c.session.Finish(stopped)
	c.session = nil

	if err != nil {
		slog.Error("experiment finish", "err", err)
		return "Experiment save error: " + err.Error()
	}
	return "Experiment done: " + summary.String()
}
