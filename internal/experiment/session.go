package experiment

import (
	"bufio"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"time"
	"webcam/internal/diagnostics"
	"webcam/internal/helpers"

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
	Backend    string // capture backend, e.g. "DirectShow"
	Width      int
	Height     int
	FPS        float64 // reported by the driver
	Format     string  // pixel format the camera actually uses
	Requested  string  // pixel format asked for, "" = driver default
	AutoLocked bool
	Controls   string
}

// part accumulates the frames of one group: all frames, or ON / OFF frames in the modulator mode.
type part struct {
	name      string // "" for all frames, "on" / "off" for the modulator
	collector *diagnostics.Collector
	level     Running
	noise     Running
	above     int

	// filled by Finish
	report diagnostics.Report
	field  Field
	done   bool
}

// suffix returns "_on" / "_off" for file names, or "" for all frames.
func (p *part) suffix() string {
	if p.name == "" {
		return ""
	}
	return "_" + p.name
}

// abovePct returns the share of frames whose temporal noise exceeded the threshold.
func (p *part) abovePct() float64 {
	if p.noise.N == 0 {
		return 0
	}
	return float64(p.above) / float64(p.noise.N) * 100
}

// Session is one running experiment: it streams per-frame data to CSV and
// accumulates per-pixel statistics for the final report.
type Session struct {
	cfg     Config
	meta    Meta
	dir     string
	started time.Time

	parts      []*part
	classifier *Classifier // modulator only
	lastState  LightState

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
	name := fmt.Sprintf("%s_%s_%dx%d@%g_%s", now.Format("2006-01-02_15-04-05"), cfg.Mode, meta.Width, meta.Height, meta.FPS, meta.Format)
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
	w.WriteString(helpers.CSVBOM)
	w.WriteString(helpers.CSVRow("frame", "t_sec", "mean", "drift", "spatial_std", "temporal_noise", "above_threshold", "state"))

	s := &Session{
		cfg:       cfg,
		meta:      meta,
		dir:       dir,
		started:   now,
		file:      file,
		csv:       w,
		lastFlush: now,
	}

	names := []string{""}
	if cfg.Mode == ModeModulator {
		names = []string{"on", "off"}
		s.classifier = &Classifier{}
	}
	for _, n := range names {
		c := diagnostics.NewCollector(0, deadThreshold, 3.0)
		c.Start(rows, cols)
		s.parts = append(s.parts, &part{name: n, collector: c})
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

// State returns the light state of the last frame (modulator mode only).
func (s *Session) State() LightState {
	return s.lastState
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

	above := sample.HasNoise && sample.Noise > s.cfg.Threshold
	if sample.HasNoise {
		s.noise.Add(sample.Noise)
		if above {
			s.above++
		}
	}

	p, state := s.partFor(sample)
	if p != nil {
		p.level.Add(sample.Mean)
		if sample.HasNoise {
			p.noise.Add(sample.Noise)
			if above {
				p.above++
			}
		}
		if gray.Rows() == s.meta.Height && gray.Cols() == s.meta.Width {
			p.collector.Add(gray)
		}
	}

	s.writeRow(sample, t, above, state)

	return t >= s.cfg.Duration()
}

// partFor picks the group the frame belongs to; nil means the frame is skipped (modulator transitions).
func (s *Session) partFor(sample Sample) (*part, string) {
	if s.classifier == nil {
		return s.parts[0], "-"
	}

	state := s.classifier.Classify(sample.Mean, sample.Time)
	prev := s.lastState
	s.lastState = state

	// the noise of a frame is measured against the previous one, so both must be in the same stable state
	if state == prev {
		switch state {
		case StateOn:
			return s.parts[0], state.String()
		case StateOff:
			return s.parts[1], state.String()
		}
	}
	return nil, state.String()
}

// writeRow appends one CSV row and flushes to disk once a second.
func (s *Session) writeRow(sample Sample, t time.Duration, above bool, state string) {
	noise, aboveFlag := "", "0"
	if sample.HasNoise {
		noise = helpers.CSVFloat(sample.Noise, 4)
	}
	if above {
		aboveFlag = "1"
	}

	s.csv.WriteString(helpers.CSVRow(
		strconv.Itoa(s.frames),
		helpers.CSVFloat(t.Seconds(), 3),
		helpers.CSVFloat(sample.Mean, 4),
		helpers.CSVFloat(sample.Mean-s.firstLevel, 4),
		helpers.CSVFloat(sample.SpatialStd, 4),
		noise,
		aboveFlag,
		state,
	))

	if time.Since(s.lastFlush) >= flushInterval {
		if err := s.csv.Flush(); err != nil {
			slog.Error("experiment csv flush", "err", err)
		}
		s.lastFlush = time.Now()
	}
}

// effectiveFPS returns the frame rate actually received.
func (s *Session) effectiveFPS() float64 {
	if span := s.prevTime.Sub(s.firstTime); span > 0 {
		return float64(s.frames-1) / span.Seconds()
	}
	return 0
}

// Summary is the short result shown on screen, in the console and in the mode matrix table.
// In the modulator mode Level, AvgNoise and the field values describe the ON frames.
type Summary struct {
	Dir       string
	Mode      Mode
	Width     int
	Height    int
	FPS       float64 // reported by the driver
	EffFPS    float64 // actually received
	Format    string
	Frames    int
	Elapsed   time.Duration
	Threshold float64
	Drift     float64

	Level         float64
	AvgNoise      float64 // temporal noise, average over frames
	AbovePct      float64
	PixelNoise    float64 // average per-pixel temporal stddev
	NonUniformity float64
	RowFPN        float64
	ColFPN        float64

	OffLevel float64 // modulator only
	OffNoise float64
	Signal   float64 // ON - OFF level
	SNR      float64 // Signal / ON temporal noise
	Switches int
	Period   float64 // seconds
}

// String formats the summary for notifications.
func (s Summary) String() string {
	if s.Mode == ModeModulator {
		return fmt.Sprintf("signal %.2f DN, SNR %.1f, noise ON %.3f / OFF %.3f DN, %d switches",
			s.Signal, s.SNR, s.AvgNoise, s.OffNoise, s.Switches)
	}
	return fmt.Sprintf("noise %.3f DN (thr %.1f, above %.1f%%), drift %+.2f DN, %d frames",
		s.AvgNoise, s.Threshold, s.AbovePct, s.Drift, s.Frames)
}

// Finish closes data.csv, computes field statistics and writes report.txt and the maps.
// stopped tells whether the user ended the run before the selected duration.
func (s *Session) Finish(stopped bool) (Summary, error) {
	elapsed := s.Elapsed()

	flushErr := s.csv.Flush()
	closeErr := s.file.Close()

	for _, p := range s.parts {
		if collected, _ := p.collector.Progress(); collected == 0 {
			p.collector.Discard()
			continue
		}
		p.report = p.collector.Finish()
		p.field = analyzeField(p.report, s.cfg.Threshold)
		p.done = true
		s.saveMaps(p)
	}
	defer func() {
		for _, p := range s.parts {
			if p.done {
				p.report.Close()
			}
		}
	}()

	summary := s.summary(elapsed)

	if flushErr != nil || closeErr != nil {
		return summary, fmt.Errorf("save data.csv: flush=%v close=%v", flushErr, closeErr)
	}

	text := s.reportText(elapsed, stopped)
	if err := os.WriteFile(filepath.Join(s.dir, "report.txt"), []byte(text), 0644); err != nil {
		return summary, fmt.Errorf("write report: %w", err)
	}

	if !s.parts[0].done {
		return summary, fmt.Errorf("no usable frames recorded")
	}

	slog.Info("experiment finished", "dir", s.dir, "frames", s.frames, "summary", summary.String())
	fmt.Printf("\nExperiment finished (%s): %s\n  Report: %s\n", FormatDuration(elapsed), summary.String(), filepath.Join(s.dir, "report.txt"))

	return summary, nil
}

// summary collects the key numbers of the run.
func (s *Session) summary(elapsed time.Duration) Summary {
	primary := s.parts[0]

	sum := Summary{
		Dir:       s.dir,
		Mode:      s.cfg.Mode,
		Width:     s.meta.Width,
		Height:    s.meta.Height,
		FPS:       s.meta.FPS,
		EffFPS:    s.effectiveFPS(),
		Format:    s.meta.Format,
		Frames:    s.frames,
		Elapsed:   elapsed,
		Threshold: s.cfg.Threshold,
		Drift:     s.lastLevel - s.firstLevel,
		Level:     primary.level.Mean(),
		AvgNoise:  primary.noise.Mean(),
		AbovePct:  primary.abovePct(),
	}

	if primary.done {
		sum.PixelNoise = primary.field.AvgNoise
		sum.NonUniformity = primary.field.NonUniformity
		sum.RowFPN = primary.field.RowFPN
		sum.ColFPN = primary.field.ColFPN
	}

	if s.classifier != nil {
		off := s.parts[1]
		sum.OffLevel = off.level.Mean()
		sum.OffNoise = off.noise.Mean()
		sum.Signal = sum.Level - sum.OffLevel
		if sum.AvgNoise > 0 {
			sum.SNR = sum.Signal / sum.AvgNoise
		}
		sum.Switches = s.classifier.Switches
		sum.Period = s.classifier.Period()
	}

	return sum
}

// saveMaps writes the average frame, the noise heat map and the defect map of one part.
func (s *Session) saveMaps(p *part) {
	path := func(name string) string {
		return filepath.Join(s.dir, name+p.suffix()+".png")
	}

	errs := []error{
		diagnostics.SaveMeanMap(p.report, path("mean")),
		diagnostics.SaveNoiseMap(p.report, path("noise_map"), max(3*p.field.AvgNoise, s.cfg.Threshold)),
		diagnostics.SaveDefectMap(p.report, path("defects")),
	}
	for _, err := range errs {
		if err != nil {
			slog.Error("experiment map save", "err", err)
		}
	}
}
