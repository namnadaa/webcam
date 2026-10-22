package experiment

import (
	"fmt"
	"math"
	"strings"
	"time"
)

// Warning limits used to flag a questionable setup in the report.
const (
	darkLevelWarning    = 20  // average level in the dark above this suggests light leaks
	saturatedPctWarning = 0.1 // % of saturated pixels in light mode
	clippedPctWarning   = 5   // % of never-changing pixels in the dark
)

// metaText describes the capture conditions; written at start so it survives a crash.
func metaText(cfg Config, meta Meta, started time.Time) string {
	var b strings.Builder

	fmt.Fprintf(&b, "Started:          %s\n", started.Format("2006-01-02 15:04:05"))
	fmt.Fprintf(&b, "Mode:             %s (%s)\n", cfg.Mode, cfg.Mode.Hint())
	fmt.Fprintf(&b, "Planned duration: %s\n", FormatDuration(cfg.Duration()))
	fmt.Fprintf(&b, "Noise threshold:  %.1f DN\n", cfg.Threshold)
	fmt.Fprintf(&b, "Camera:           %s\n", meta.Camera)
	fmt.Fprintf(&b, "Resolution:       %dx%d\n", meta.Width, meta.Height)
	fmt.Fprintf(&b, "FPS (driver):     %.2f\n", meta.FPS)
	fmt.Fprintf(&b, "Format:           %s (requested: %s)\n", meta.Format, requestedText(meta.Requested))
	fmt.Fprintf(&b, "Source:           raw camera frame, grayscale (filters bypassed)\n")
	fmt.Fprintf(&b, "Auto controls:    %s\n", lockedText(meta.AutoLocked))
	fmt.Fprintf(&b, "Controls:         %s\n", meta.Controls)

	return b.String()
}

// reportText builds the final report: capture conditions, behaviour over time and across the field.
func (s *Session) reportText(elapsed time.Duration, stopped bool) string {
	var b strings.Builder

	fmt.Fprintf(&b, "EXPERIMENT REPORT\n=================\n")
	b.WriteString(metaText(s.cfg, s.meta, s.started))

	status := "completed"
	if stopped {
		status = "stopped by user"
	}
	fmt.Fprintf(&b, "Actual duration:  %s (%s)\n", FormatDuration(elapsed), status)
	fmt.Fprintf(&b, "Frames:           %d (effective %.2f FPS)\n", s.frames, s.effectiveFPS())
	fmt.Fprintf(&b, "Frame interval:   avg %.1f ms, min %.1f ms, max %.1f ms, late frames %d\n",
		s.interval.Mean(), s.interval.Min, s.interval.Max, s.late)

	fmt.Fprintf(&b, "\nOVER TIME (per-frame values, data.csv)\n--------------------------------------\n")
	fmt.Fprintf(&b, "Level:            avg %.3f, min %.3f, max %.3f, stddev %.4f DN\n",
		s.level.Mean(), s.level.Min, s.level.Max, s.level.StdDev())
	fmt.Fprintf(&b, "Drift:            %+.3f DN (last - first), trend %+.4f DN/min\n",
		s.lastLevel-s.firstLevel, s.trend.Slope())
	fmt.Fprintf(&b, "Temporal noise:   avg %.4f, min %.4f, max %.4f, stddev %.4f DN\n",
		s.noise.Mean(), s.noise.Min, s.noise.Max, s.noise.StdDev())

	abovePct := 0.0
	if s.noise.N > 0 {
		abovePct = float64(s.above) / float64(s.noise.N) * 100
	}
	fmt.Fprintf(&b, "Above threshold:  %d of %d frames (%.2f%%)\n", s.above, s.noise.N, abovePct)
	fmt.Fprintf(&b, "Spatial spread:   avg %.4f DN per frame\n", s.spatial.Mean())

	if s.classifier != nil {
		s.writeModulation(&b)
	}

	for _, p := range s.parts {
		s.writeField(&b, p)
	}

	if warnings := s.warnings(); len(warnings) > 0 {
		fmt.Fprintf(&b, "\nWARNINGS\n--------\n")
		for _, w := range warnings {
			fmt.Fprintf(&b, "- %s\n", w)
		}
	}

	b.WriteString(`
DEFINITIONS
-----------
DN:               digital number, one step of the 8-bit pixel value (0..255)
Temporal noise:   per frame, stddev(frame - previous frame) / sqrt(2); uniform brightness changes (flicker) cancel out
Pixel noise:      per pixel, stddev over the whole run; includes flicker and drift
Non-uniformity:   stddev of per-pixel averages (fixed pattern across the field)
Row/Column FPN:   stddev of row/column averages (stripes)
`)
	if s.classifier != nil {
		b.WriteString(`Signal:           average ON level - average OFF level
SNR:              signal / ON temporal noise
Light state:      ON/OFF by average level with hysteresis; frames right after a switch are excluded
`)
	}

	b.WriteString(`
FILES: data.csv (per frame), mean*.png (average frame), noise_map*.png (per-pixel noise, brighter = noisier),
       defects*.png (blue = never changing, red = noisy); _on/_off suffix = light state in the modulator mode
`)

	return b.String()
}

// writeModulation describes the ON/OFF cycles and the signal.
func (s *Session) writeModulation(b *strings.Builder) {
	c := s.classifier
	on, off := s.parts[0], s.parts[1]

	fmt.Fprintf(b, "\nMODULATION (light ON / OFF)\n---------------------------\n")
	fmt.Fprintf(b, "Switches:         %d\n", c.Switches)
	fmt.Fprintf(b, "ON segments:      %d, avg %.2f s, min %.2f s, max %.2f s\n", c.OnSegment.N, c.OnSegment.Mean(), c.OnSegment.Min, c.OnSegment.Max)
	fmt.Fprintf(b, "OFF segments:     %d, avg %.2f s, min %.2f s, max %.2f s\n", c.OffSegment.N, c.OffSegment.Mean(), c.OffSegment.Min, c.OffSegment.Max)
	if period := c.Period(); period > 0 {
		fmt.Fprintf(b, "Period:           %.2f s (%.3f Hz)\n", period, 1/period)
	}
	fmt.Fprintf(b, "ON frames:        %d, level %.3f DN (stddev %.4f), temporal noise %.4f DN, above threshold %.2f%%\n",
		on.level.N, on.level.Mean(), on.level.StdDev(), on.noise.Mean(), on.abovePct())
	fmt.Fprintf(b, "OFF frames:       %d, level %.3f DN (stddev %.4f), temporal noise %.4f DN, above threshold %.2f%%\n",
		off.level.N, off.level.Mean(), off.level.StdDev(), off.noise.Mean(), off.abovePct())

	signal := on.level.Mean() - off.level.Mean()
	fmt.Fprintf(b, "Signal:           %.3f DN\n", signal)
	if n := on.noise.Mean(); n > 0 {
		fmt.Fprintf(b, "SNR:              %.2f (%.1f dB)\n", signal/n, 20*math.Log10(math.Abs(signal/n)))
	}
}

// writeField describes the per-pixel statistics of one part.
func (s *Session) writeField(b *strings.Builder, p *part) {
	title := "ACROSS THE FIELD (per-pixel statistics over the run)"
	if p.name != "" {
		title = fmt.Sprintf("ACROSS THE FIELD, LIGHT %s (per-pixel statistics over %s frames)", strings.ToUpper(p.name), p.name)
	}
	fmt.Fprintf(b, "\n%s\n%s\n", title, strings.Repeat("-", len(title)))

	if !p.done {
		b.WriteString("No frames recorded.\n")
		return
	}

	r, f := p.report, p.field
	total := float64(r.TotalPixels)

	fmt.Fprintf(b, "Frames:           %d\n", r.Frames)
	fmt.Fprintf(b, "Mean level:       %.3f DN\n", f.MeanLevel)
	fmt.Fprintf(b, "Non-uniformity:   %.4f DN (%s)\n", f.NonUniformity, nonUniformityName(s.cfg.Mode, p.name, f))
	fmt.Fprintf(b, "Row FPN:          %.4f DN\n", f.RowFPN)
	fmt.Fprintf(b, "Column FPN:       %.4f DN\n", f.ColFPN)
	fmt.Fprintf(b, "Pixel noise:      avg %.4f DN, max %.4f DN\n", f.AvgNoise, f.MaxNoise)
	fmt.Fprintf(b, "Above threshold:  %d pixels (%.4f%%)\n", f.AboveThresh, float64(f.AboveThresh)/total*100)
	fmt.Fprintf(b, "Never changing:   %d pixels (%.4f%%, stddev < %.1f DN: dead, stuck or clipped)\n", r.DeadCount, float64(r.DeadCount)/total*100, deadThreshold)
	fmt.Fprintf(b, "Noisy (>3x avg):  %d pixels (%.4f%%)\n", r.NoisyCount, float64(r.NoisyCount)/total*100)
	fmt.Fprintf(b, "Saturated:        %d pixels (%.4f%%)\n", f.Saturated, float64(f.Saturated)/total*100)

	zoneMin, zoneMax := f.ZoneSpread()
	fmt.Fprintf(b, "\nZones %dx%d: level DN / noise DN (zone spread %.3f..%.3f)\n", ZoneCols, ZoneRows, zoneMin, zoneMax)
	for _, row := range f.Zones {
		for i, z := range row {
			if i > 0 {
				b.WriteString(" | ")
			}
			fmt.Fprintf(b, "%6.2f/%5.3f", z.Mean, z.Noise)
		}
		b.WriteString("\n")
	}
}

// warnings lists setup problems that make the result questionable.
func (s *Session) warnings() []string {
	var w []string

	if !s.meta.AutoLocked {
		w = append(w, "auto controls were not locked: the camera may have changed exposure/gain during the run")
	}

	if primary := s.parts[0]; primary.done {
		f, r := primary.field, primary.report
		total := float64(r.TotalPixels)

		if s.cfg.Mode == ModeDark && f.MeanLevel > darkLevelWarning {
			w = append(w, fmt.Sprintf("dark level %.1f DN is high: the lens may not be fully covered", f.MeanLevel))
		}
		if s.cfg.Mode == ModeDark && float64(r.DeadCount)/total*100 > clippedPctWarning {
			w = append(w, fmt.Sprintf("%.1f%% of pixels never changed: the black level is probably clipped at 0, dark noise is underestimated", float64(r.DeadCount)/total*100))
		}
		if (s.cfg.Mode == ModeLight || s.cfg.Mode == ModeModulator) && float64(f.Saturated)/total*100 > saturatedPctWarning {
			w = append(w, "saturated pixels present: reduce the light, noise in saturated areas is underestimated")
		}
	}

	if c := s.classifier; c != nil {
		if c.Contrast() < minContrast {
			w = append(w, fmt.Sprintf("ON and OFF levels differ by less than %.0f DN: the light change was not detected", minContrast))
		} else if c.Switches < 2 {
			w = append(w, "fewer than 2 switches: not enough ON/OFF cycles for a reliable comparison")
		}
	}

	if s.late > 0 && s.frames > 0 && float64(s.late)/float64(s.frames) > 0.05 {
		w = append(w, fmt.Sprintf("%d late frames: processing could not keep up with the camera FPS", s.late))
	}

	return w
}

// nonUniformityName explains what the non-uniformity number means in the given mode and light state.
func nonUniformityName(m Mode, state string, f Field) string {
	dark := m == ModeDark || (m == ModeModulator && state == "off")
	light := m == ModeLight || (m == ModeModulator && state == "on")

	switch {
	case dark:
		return "DSNU, dark signal non-uniformity"
	case light:
		if f.MeanLevel > 0 {
			return fmt.Sprintf("PRNU, %.3f%% of level", f.NonUniformity/f.MeanLevel*100)
		}
		return "PRNU"
	}
	return "includes scene content"
}

// requestedText shows an empty requested format as the driver default.
func requestedText(format string) string {
	if format == "" {
		return "auto"
	}
	return format
}

// lockedText describes whether auto exposure/white balance/focus were locked.
func lockedText(locked bool) string {
	if locked {
		return "locked"
	}
	return "NOT locked (press L before the run)"
}
