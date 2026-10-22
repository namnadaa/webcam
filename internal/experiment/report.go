package experiment

import (
	"fmt"
	"strings"
	"time"
	"webcam/internal/diagnostics"
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
	fmt.Fprintf(&b, "Format:           %s\n", meta.Format)
	fmt.Fprintf(&b, "Source:           raw camera frame, grayscale (filters bypassed)\n")
	fmt.Fprintf(&b, "Auto controls:    %s\n", lockedText(meta.AutoLocked))
	fmt.Fprintf(&b, "Controls:         %s\n", meta.Controls)

	return b.String()
}

// reportText builds the final report: capture conditions, behaviour over time and across the field.
func (s *Session) reportText(r diagnostics.Report, f Field, elapsed time.Duration, stopped bool) string {
	var b strings.Builder

	fmt.Fprintf(&b, "EXPERIMENT REPORT\n=================\n")
	b.WriteString(metaText(s.cfg, s.meta, s.started))

	status := "completed"
	if stopped {
		status = "stopped by user"
	}
	fmt.Fprintf(&b, "Actual duration:  %s (%s)\n", FormatDuration(elapsed), status)

	effFPS := 0.0
	if span := s.prevTime.Sub(s.firstTime); span > 0 {
		effFPS = float64(s.frames-1) / span.Seconds()
	}
	fmt.Fprintf(&b, "Frames:           %d (effective %.2f FPS)\n", s.frames, effFPS)
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

	total := float64(r.TotalPixels)
	fmt.Fprintf(&b, "\nACROSS THE FIELD (per-pixel statistics over the run)\n----------------------------------------------------\n")
	fmt.Fprintf(&b, "Mean level:       %.3f DN\n", f.MeanLevel)
	fmt.Fprintf(&b, "Non-uniformity:   %.4f DN (%s)\n", f.NonUniformity, nonUniformityName(s.cfg.Mode, f))
	fmt.Fprintf(&b, "Row FPN:          %.4f DN\n", f.RowFPN)
	fmt.Fprintf(&b, "Column FPN:       %.4f DN\n", f.ColFPN)
	fmt.Fprintf(&b, "Pixel noise:      avg %.4f DN, max %.4f DN\n", f.AvgNoise, f.MaxNoise)
	fmt.Fprintf(&b, "Above threshold:  %d pixels (%.4f%%)\n", f.AboveThresh, float64(f.AboveThresh)/total*100)
	fmt.Fprintf(&b, "Never changing:   %d pixels (%.4f%%, stddev < %.1f DN: dead, stuck or clipped)\n", r.DeadCount, float64(r.DeadCount)/total*100, deadThreshold)
	fmt.Fprintf(&b, "Noisy (>3x avg):  %d pixels (%.4f%%)\n", r.NoisyCount, float64(r.NoisyCount)/total*100)
	fmt.Fprintf(&b, "Saturated:        %d pixels (%.4f%%)\n", f.Saturated, float64(f.Saturated)/total*100)

	zoneMin, zoneMax := f.ZoneSpread()
	fmt.Fprintf(&b, "\nZONES %dx%d: level DN / noise DN (zone spread %.3f..%.3f)\n", ZoneCols, ZoneRows, zoneMin, zoneMax)
	for _, row := range f.Zones {
		for i, z := range row {
			if i > 0 {
				b.WriteString(" | ")
			}
			fmt.Fprintf(&b, "%6.2f/%5.3f", z.Mean, z.Noise)
		}
		b.WriteString("\n")
	}

	if warnings := s.warnings(r, f, total); len(warnings) > 0 {
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

FILES: data.csv (per frame), mean.png (average frame), noise_map.png (per-pixel noise, brighter = noisier),
       defects.png (blue = never changing, red = noisy)
`)

	return b.String()
}

// warnings lists setup problems that make the result questionable.
func (s *Session) warnings(r diagnostics.Report, f Field, total float64) []string {
	var w []string

	if !s.meta.AutoLocked {
		w = append(w, "auto controls were not locked: the camera may have changed exposure/gain during the run")
	}
	if s.cfg.Mode == ModeDark && f.MeanLevel > darkLevelWarning {
		w = append(w, fmt.Sprintf("dark level %.1f DN is high: the lens may not be fully covered", f.MeanLevel))
	}
	if s.cfg.Mode == ModeDark && float64(r.DeadCount)/total*100 > clippedPctWarning {
		w = append(w, fmt.Sprintf("%.1f%% of pixels never changed: the black level is probably clipped at 0, dark noise is underestimated", float64(r.DeadCount)/total*100))
	}
	if s.cfg.Mode == ModeLight && float64(f.Saturated)/total*100 > saturatedPctWarning {
		w = append(w, "saturated pixels present: reduce the light, noise in saturated areas is underestimated")
	}
	if s.late > 0 && s.frames > 0 && float64(s.late)/float64(s.frames) > 0.05 {
		w = append(w, fmt.Sprintf("%d late frames: processing could not keep up with the camera FPS", s.late))
	}

	return w
}

// nonUniformityName explains what the non-uniformity number means in the given mode.
func nonUniformityName(m Mode, f Field) string {
	switch m {
	case ModeDark:
		return "DSNU, dark signal non-uniformity"
	case ModeLight:
		if f.MeanLevel > 0 {
			return fmt.Sprintf("PRNU, %.3f%% of level", f.NonUniformity/f.MeanLevel*100)
		}
		return "PRNU"
	}
	return "includes scene content"
}

// lockedText describes whether auto exposure/white balance/focus were locked.
func lockedText(locked bool) string {
	if locked {
		return "locked"
	}
	return "NOT locked (press L before the run)"
}
