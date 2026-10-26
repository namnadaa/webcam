package experiment

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"webcam/internal/helpers"
	"webcam/internal/i18n"
)

// Warmup is how long the camera runs after a mode switch before the measurement starts,
// so the stream and any automatic adjustment can settle.
const Warmup = 3 * time.Second

// Step is one camera mode of the matrix.
type Step struct {
	Width  int
	Height int
	FPS    float64
	Format string // pixel format to request, "" = driver default
}

// String returns the mode as "1280x720@30" or "1280x720@30 MJPG".
func (s Step) String() string {
	if s.Format != "" {
		return fmt.Sprintf("%dx%d@%g %s", s.Width, s.Height, s.FPS, s.Format)
	}
	return fmt.Sprintf("%dx%d@%g", s.Width, s.Height, s.FPS)
}

// Matrix runs the same experiment in several camera modes one after another
// and collects the results into one comparison table.
type Matrix struct {
	steps   []Step
	idx     int
	cfg     Config
	started time.Time
	results []Summary
}

// NewMatrix prepares a run over the given modes with the experiment settings cfg.
func NewMatrix(steps []Step, cfg Config) *Matrix {
	return &Matrix{
		steps:   steps,
		cfg:     cfg,
		started: time.Now(),
	}
}

// Config returns the experiment settings used for every step.
func (m *Matrix) Config() Config {
	return m.cfg
}

// Current returns the camera mode being measured.
func (m *Matrix) Current() Step {
	return m.steps[m.idx]
}

// Progress returns "step/total", e.g. "2/6".
func (m *Matrix) Progress() string {
	return fmt.Sprintf("%d/%d", m.idx+1, len(m.steps))
}

// Record stores the result of the current step and moves to the next one.
// Returns false when all steps are done.
func (m *Matrix) Record(s Summary) bool {
	m.results = append(m.results, s)
	m.idx++
	return m.idx < len(m.steps)
}

// Save writes summary.csv and summary.txt with one row per measured mode.
// aborted marks a run stopped by the user.
func (m *Matrix) Save(aborted bool) (string, error) {
	name := fmt.Sprintf("%s_matrix_%s", m.started.Format("2006-01-02_15-04-05"), m.cfg.Mode)
	dir := filepath.Join(RootDir, name)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("create matrix dir: %w", err)
	}

	if err := os.WriteFile(filepath.Join(dir, "summary.csv"), []byte(m.csv()), 0644); err != nil {
		return "", fmt.Errorf("write summary.csv: %w", err)
	}

	txtPath := filepath.Join(dir, "summary.txt")
	if err := os.WriteFile(txtPath, []byte(m.text(aborted)), 0644); err != nil {
		return "", fmt.Errorf("write summary.txt: %w", err)
	}

	slog.Info("mode matrix saved", "dir", dir, "steps", len(m.results), "aborted", aborted)
	fmt.Print(i18n.Tf("\nMode matrix finished: %d of %d modes measured\n  Summary: %s\n", len(m.results), len(m.steps), txtPath))

	return txtPath, nil
}

// csv formats all results as a machine-readable table.
func (m *Matrix) csv() string {
	var b strings.Builder

	b.WriteString(helpers.CSVBOM)
	b.WriteString(helpers.CSVRow("requested", "width", "height", "format", "fps_driver", "fps_effective", "frames", "level",
		"temporal_noise", "above_threshold_pct", "pixel_noise", "non_uniformity", "row_fpn", "col_fpn", "drift",
		"off_level", "off_noise", "signal", "snr", "switches", "period_s", "dir"))

	f := helpers.CSVFloat
	for i, r := range m.results {
		b.WriteString(helpers.CSVRow(
			m.steps[i].String(), strconv.Itoa(r.Width), strconv.Itoa(r.Height), r.Format,
			f(r.FPS, 2), f(r.EffFPS, 2), strconv.Itoa(r.Frames), f(r.Level, 4), f(r.AvgNoise, 4), f(r.AbovePct, 2),
			f(r.PixelNoise, 4), f(r.NonUniformity, 4), f(r.RowFPN, 4), f(r.ColFPN, 4), f(r.Drift, 4),
			f(r.OffLevel, 4), f(r.OffNoise, 4), f(r.Signal, 4), f(r.SNR, 2), strconv.Itoa(r.Switches), f(r.Period, 3),
			r.Dir,
		))
	}

	return b.String()
}

// text formats the results as a table for reading.
func (m *Matrix) text(aborted bool) string {
	var b strings.Builder

	status := i18n.T("completed")
	if aborted {
		status = i18n.T("aborted by user")
	}

	b.WriteString(i18n.T("MODE MATRIX\n===========\n"))
	b.WriteString(i18n.Tf("Started:          %s\n", m.started.Format("2006-01-02 15:04:05")))
	b.WriteString(i18n.Tf("Experiment:       %s\n", m.cfg.Text()))
	b.WriteString(i18n.Tf("Modes:            %d of %d measured (%s)\n", len(m.results), len(m.steps), status))
	b.WriteString(i18n.Tf("Warmup per mode:  %s\n\n", FormatDuration(Warmup)))

	fmt.Fprintf(&b, "%-16s %-11s %-7s %7s %7s %8s %8s %8s %8s %8s %8s %8s",
		i18n.T("requested"), i18n.T("actual"), i18n.T("format"), "fps", i18n.T("eff.fps"), i18n.T("level"), i18n.T("t.noise"),
		i18n.T("px.noise"), i18n.T("nonunif"), i18n.T("row.fpn"), i18n.T("col.fpn"), i18n.T("drift"))
	if m.cfg.Mode == ModeModulator {
		fmt.Fprintf(&b, " %8s %8s %7s", i18n.T("signal"), i18n.T("off.nse"), "SNR")
	}
	b.WriteString("\n")

	for i, r := range m.results {
		fmt.Fprintf(&b, "%-16s %-11s %-7s %7.2f %7.2f %8.3f %8.4f %8.4f %8.4f %8.4f %8.4f %+8.3f",
			m.steps[i], fmt.Sprintf("%dx%d", r.Width, r.Height), r.Format, r.FPS, r.EffFPS, r.Level, r.AvgNoise,
			r.PixelNoise, r.NonUniformity, r.RowFPN, r.ColFPN, r.Drift)
		if m.cfg.Mode == ModeModulator {
			fmt.Fprintf(&b, " %8.3f %8.4f %7.2f", r.Signal, r.OffNoise, r.SNR)
		}
		b.WriteString("\n")
	}

	b.WriteString(i18n.T("\nAll values in DN. t.noise = temporal noise per frame, px.noise = per-pixel stddev over the run.\n"))
	if m.cfg.Mode == ModeModulator {
		b.WriteString(i18n.T("Modulator: level, noise and field values are for light ON frames.\n"))
	}
	b.WriteString(i18n.T("Detailed report of every mode: see the dir column in summary.csv.\n"))

	return b.String()
}
