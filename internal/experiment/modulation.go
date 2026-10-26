package experiment

import (
	"math"
	"time"
	"webcam/internal/i18n"
)

// LightState is the classification of a frame in the modulator mode.
type LightState int

const (
	StateUnknown    LightState = iota // ON and OFF levels are not separated yet
	StateTransition                   // light is switching or the level is still settling
	StateOff
	StateOn
)

// String returns the state name used in CSV and reports.
func (s LightState) String() string {
	switch s {
	case StateTransition:
		return "transition"
	case StateOff:
		return "off"
	case StateOn:
		return "on"
	}
	return "unknown"
}

// Title returns the state name in the interface language.
func (s LightState) Title() string {
	switch s {
	case StateTransition:
		return i18n.T("transition")
	case StateOff:
		return i18n.T("off")
	case StateOn:
		return i18n.T("on")
	}
	return i18n.T("unknown")
}

// Classifier tuning.
const (
	minContrast  = 5.0 // DN between OFF and ON levels needed to tell them apart
	hysteresis   = 0.2 // switching happens at 50% +- 20% of the ON/OFF contrast
	settleFrames = 2   // frames skipped after every switch
	settleStep   = 0.1 // a frame whose level moved more than this share of the contrast is still settling
)

// Classifier tells light ON frames from OFF frames by the average level.
// It learns the OFF and ON levels from the darkest and brightest frames seen so far
// and switches with hysteresis, so noise near the middle does not cause false switches.
type Classifier struct {
	minLevel float64
	maxLevel float64
	seen     bool

	state    LightState // last stable state, StateOn or StateOff once known
	settle   int
	prevMean float64

	Switches   int
	segStart   time.Time
	segCounted bool // the current segment started with a switch, so its length is complete
	OnSegment  Running
	OffSegment Running
}

// Contrast returns the difference between the brightest and darkest levels seen.
func (c *Classifier) Contrast() float64 {
	return c.maxLevel - c.minLevel
}

// Classify assigns a state to a frame with the given average level.
func (c *Classifier) Classify(mean float64, t time.Time) LightState {
	prev := c.prevMean
	hasPrev := c.seen
	c.prevMean = mean

	if !c.seen {
		c.minLevel, c.maxLevel, c.seen = mean, mean, true
	}
	c.minLevel = math.Min(c.minLevel, mean)
	c.maxLevel = math.Max(c.maxLevel, mean)

	contrast := c.Contrast()
	if contrast < minContrast {
		return StateUnknown
	}

	next := c.state
	switch {
	case mean > c.minLevel+(0.5+hysteresis)*contrast:
		next = StateOn
	case mean < c.minLevel+(0.5-hysteresis)*contrast:
		next = StateOff
	}

	if next == StateUnknown {
		return StateTransition
	}

	if next != c.state {
		c.switchTo(next, t)
	}

	if c.settle > 0 {
		c.settle--
		return StateTransition
	}
	if hasPrev && math.Abs(mean-prev) > settleStep*contrast {
		return StateTransition
	}

	return c.state
}

// switchTo records the end of the current segment and starts a new one.
func (c *Classifier) switchTo(next LightState, t time.Time) {
	if c.state != StateUnknown {
		c.Switches++
		if c.segCounted {
			seconds := t.Sub(c.segStart).Seconds()
			if c.state == StateOn {
				c.OnSegment.Add(seconds)
			} else {
				c.OffSegment.Add(seconds)
			}
		}
		c.segCounted = true
	}

	c.state = next
	c.segStart = t
	c.settle = settleFrames
}

// Period returns the average full ON+OFF cycle length, or 0 if it is not known yet.
func (c *Classifier) Period() float64 {
	if c.OnSegment.N == 0 || c.OffSegment.N == 0 {
		return 0
	}
	return c.OnSegment.Mean() + c.OffSegment.Mean()
}
