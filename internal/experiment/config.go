package experiment

import (
	"fmt"
	"math"
	"time"
)

// Mode is the measurement scenario; it defines setup hints and how the report is interpreted.
type Mode int

const (
	ModeDark      Mode = iota // lens covered: dark noise and dark signal non-uniformity
	ModeLight                 // uniform illumination: shot noise and photo response non-uniformity
	ModeStability             // fixed scene: drift of level and noise over time and across the field
	modeCount
)

// String returns the short mode name used in folder names and the UI.
func (m Mode) String() string {
	switch m {
	case ModeDark:
		return "dark"
	case ModeLight:
		return "light"
	case ModeStability:
		return "stability"
	}
	return "unknown"
}

// Hint tells the user how to prepare the camera for the mode.
func (m Mode) Hint() string {
	switch m {
	case ModeDark:
		return "cover the lens completely"
	case ModeLight:
		return "point at a uniform light source, avoid saturation"
	case ModeStability:
		return "fixed camera and static scene"
	}
	return ""
}

// Next returns the following mode, cycling back to the first.
func (m Mode) Next() Mode {
	return (m + 1) % modeCount
}

// Durations lists the selectable measurement lengths.
var Durations = []time.Duration{
	30 * time.Second,
	1 * time.Minute,
	5 * time.Minute,
	10 * time.Minute,
	30 * time.Minute,
}

// Threshold limits for the temporal noise threshold in DN (digital numbers, 0..255).
const (
	ThresholdStep = 0.1
	ThresholdMin  = 0.1
	ThresholdMax  = 50
)

// Config is what the user selects before starting an experiment.
type Config struct {
	Mode          Mode
	DurationIndex int
	Threshold     float64 // temporal noise threshold, DN
}

// DefaultConfig returns a dark 1-minute experiment with a 2 DN threshold.
func DefaultConfig() Config {
	return Config{
		Mode:          ModeDark,
		DurationIndex: 1,
		Threshold:     2.0,
	}
}

// Duration returns the selected measurement length.
func (c Config) Duration() time.Duration {
	return Durations[c.DurationIndex]
}

// NextDuration selects the following duration, cycling back to the first.
func (c *Config) NextDuration() {
	c.DurationIndex = (c.DurationIndex + 1) % len(Durations)
}

// AdjustThreshold changes the threshold by delta keeping it in the allowed range.
func (c *Config) AdjustThreshold(delta float64) {
	t := math.Round((c.Threshold+delta)*10) / 10
	c.Threshold = math.Min(ThresholdMax, math.Max(ThresholdMin, t))
}

// String formats the config for notifications and console output.
func (c Config) String() string {
	return fmt.Sprintf("mode=%s duration=%s threshold=%.1f DN", c.Mode, FormatDuration(c.Duration()), c.Threshold)
}

// FormatDuration prints durations as "30s", "5m" or "1m30s".
func FormatDuration(d time.Duration) string {
	d = d.Round(time.Second)
	if d >= time.Minute && d%time.Minute == 0 {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	return d.String()
}

// FormatClock prints durations as "mm:ss".
func FormatClock(d time.Duration) string {
	s := int(d.Round(time.Second).Seconds())
	return fmt.Sprintf("%02d:%02d", s/60, s%60)
}
