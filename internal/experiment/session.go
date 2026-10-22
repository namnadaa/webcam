package experiment

import (
	"bufio"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"
	"webcam/internal/diagnostics"

	"gocv.io/x/gocv"
)

// RootDir is where every experiment gets its own folder.
const RootDir = "experiments"

// deadThreshold is the per-pixel temporal stddev (DN) below which a pixel is treated as never changing.
const deadThreshold = 0.1

// flushInterval is how often CSV data is written to disk, so a crash loses at most this much.
const flushInterval = time.Second

// Meta describes the capture conditions the experiment was run under.
type Meta struct {
	Camera     string
	Width      int
	Height     int
	FPS        float64 // reported by the driver
	Format     string
	AutoLocked bool
	Controls   string
}

// Session is one running experiment: it streams per-frame data to CSV and
// accumulates per-pixel statistics for the final report.
type Session struct {
	cfg     Config
	meta    Meta
	dir     string
	started time.Time

	collector *diagnostics.Collector

	file      *os.File
	csv       *bufio.Writer
	lastFlush time.Time

	frames     int
	firstTime  time.Time
	firstLevel float64
	lastLevel  float64
	prevTime   time.Time
	late       int // frame intervals longer than 1.5x the nominal one

	level    Running
	spatial  Running
	noise    Running
	interval Running // ms
	trend    Trend   // level vs minutes
	above    int
}

// Start creates the experiment folder, writes meta.txt and opens data.csv.
func Start(cfg Config, meta Meta, rows, cols int) (*Session, error) {
	now := time.Now()
	name := fmt.Sprintf("%s_%s_%dx%d@%g", now.Format("2006-01-02_15-04-05"), cfg.Mode, meta.Width, meta.Height, meta.FPS)
	dir := filepath.Join(RootDir, name)

	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("create experiment dir: %w", err)
	}

	if err := os.WriteFile(filepath.Join(dir, "meta.txt"), []byte(metaText(cfg, meta, now)), 0644); err != nil {
		return nil, fmt.Errorf("write meta: %w", err)
	}

	file, err := os.Create(filepath.Join(dir, "data.csv"))
	if err != nil {
		return nil, fmt.Errorf("create data.csv: %w", err)
	}

	w := bufio.NewWriter(file)
	fmt.Fprintln(w, "frame,t_sec,mean,drift,spatial_std,temporal_noise,above_threshold")

	collector := diagnostics.NewCollector(0, deadThreshold, 3.0)
	collector.Start(rows, cols)

	s := &Session{
		cfg:       cfg,
		meta:      meta,
		dir:       dir,
		started:   now,
		collector: collector,
		file:      file,
		csv:       w,
		lastFlush: now,
	}

	slog.Info("experiment started", "dir", dir, "config", cfg.String())
	fmt.Printf("\nExperiment started: %s\n  Setup: %s\n  Noise threshold: %.1f DN\n  Output: %s\n",
		cfg.String(), cfg.Mode.Hint(), cfg.Threshold, dir)

	return s, nil
}

// Config returns the settings the session was started with.
func (s *Session) Config() Config {
	return s.cfg
}

// Elapsed returns the time since the start.
func (s *Session) Elapsed() time.Duration {
	return time.Since(s.started)
}

// Add records one frame. gray is the grayscale raw frame the sample was measured on.
// Returns true when the selected duration is over.
func (s *Session) Add(sample Sample, gray gocv.Mat) bool {
	s.frames++
	t := sample.Time.Sub(s.started)

	if s.frames == 1 {
		s.firstTime = sample.Time
		s.firstLevel = sample.Mean
	}
	s.lastLevel = sample.Mean

	s.level.Add(sample.Mean)
	s.spatial.Add(sample.SpatialStd)
	s.trend.Add(t.Minutes(), sample.Mean)

	if !s.prevTime.IsZero() {
		gap := sample.Time.Sub(s.prevTime)
		s.interval.Add(float64(gap) / float64(time.Millisecond))
		if s.meta.FPS > 0 && gap.Seconds() > 1.5/s.meta.FPS {
			s.late++
		}
	}
	s.prevTime = sample.Time

	noise := ""
	above := 0
	if sample.HasNoise {
		s.noise.Add(sample.Noise)
		noise = fmt.Sprintf("%.4f", sample.Noise)
		if sample.Noise > s.cfg.Threshold {
			s.above++
			above = 1
		}
	}

	fmt.Fprintf(s.csv, "%d,%.3f,%.4f,%.4f,%.4f,%s,%d\n",
		s.frames, t.Seconds(), sample.Mean, sample.Mean-s.firstLevel, sample.SpatialStd, noise, above)

	if time.Since(s.lastFlush) >= flushInterval {
		if err := s.csv.Flush(); err != nil {
			slog.Error("experiment csv flush", "err", err)
		}
		s.lastFlush = time.Now()
	}

	if gray.Rows() == s.meta.Height && gray.Cols() == s.meta.Width {
		s.collector.Add(gray)
	}

	return t >= s.cfg.Duration()
}

// Summary is the short result shown on screen and in the console.
type Summary struct {
	Dir       string
	Frames    int
	Elapsed   time.Duration
	AvgNoise  float64
	Threshold float64
	AbovePct  float64
	Drift     float64
}

// String formats the summary for notifications.
func (s Summary) String() string {
	return fmt.Sprintf("noise %.3f DN (thr %.1f, above %.1f%%), drift %+.2f DN, %d frames",
		s.AvgNoise, s.Threshold, s.AbovePct, s.Drift, s.Frames)
}

// Finish closes data.csv, computes field statistics and writes report.txt and the maps.
// stopped tells whether the user ended the run before the selected duration.
func (s *Session) Finish(stopped bool) (Summary, error) {
	elapsed := s.Elapsed()

	flushErr := s.csv.Flush()
	closeErr := s.file.Close()

	summary := Summary{
		Dir:       s.dir,
		Frames:    s.frames,
		Elapsed:   elapsed,
		AvgNoise:  s.noise.Mean(),
		Threshold: s.cfg.Threshold,
		Drift:     s.lastLevel - s.firstLevel,
	}
	if s.noise.N > 0 {
		summary.AbovePct = float64(s.above) / float64(s.noise.N) * 100
	}

	if flushErr != nil || closeErr != nil {
		return summary, fmt.Errorf("save data.csv: flush=%v close=%v", flushErr, closeErr)
	}

	if collected, _ := s.collector.Progress(); collected == 0 {
		s.collector.Discard()
		return summary, fmt.Errorf("no frames recorded")
	}

	report := s.collector.Finish()
	defer report.Close()

	field := analyzeField(report, s.cfg.Threshold)

	maps := []struct {
		name string
		save func() error
	}{
		{"mean.png", func() error { return diagnostics.SaveMeanMap(report, filepath.Join(s.dir, "mean.png")) }},
		{"noise_map.png", func() error {
			return diagnostics.SaveNoiseMap(report, filepath.Join(s.dir, "noise_map.png"), max(3*field.AvgNoise, s.cfg.Threshold))
		}},
		{"defects.png", func() error { return diagnostics.SaveDefectMap(report, filepath.Join(s.dir, "defects.png")) }},
	}
	for _, m := range maps {
		if err := m.save(); err != nil {
			slog.Error("experiment map save", "file", m.name, "err", err)
		}
	}

	text := s.reportText(report, field, elapsed, stopped)
	if err := os.WriteFile(filepath.Join(s.dir, "report.txt"), []byte(text), 0644); err != nil {
		return summary, fmt.Errorf("write report: %w", err)
	}

	slog.Info("experiment finished", "dir", s.dir, "frames", s.frames, "avg_noise", summary.AvgNoise, "drift", summary.Drift)
	fmt.Printf("\nExperiment finished (%s): %s\n  Report: %s\n", FormatDuration(elapsed), summary.String(), filepath.Join(s.dir, "report.txt"))

	return summary, nil
}
