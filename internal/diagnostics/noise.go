package diagnostics

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"
	"webcam/internal/media"

	"gocv.io/x/gocv"
)

// Report holds the result of a per-pixel noise / dead-pixel scan.
type Report struct {
	Mean        gocv.Mat // average brightness per pixel (CV64F)
	StdDev      gocv.Mat // standard deviation per pixel (CV64F)
	DeadMask    gocv.Mat // 255 where a pixel never changes (stuck/dead)
	NoisyMask   gocv.Mat // 255 where a pixel is an outlier by noise level
	DeadCount   int
	NoisyCount  int
	TotalPixels int
	MeanLevel   float64 // average brightness over the frame
	AvgStdDev   float64 // average temporal noise over the frame
	Frames      int
	Duration    time.Duration
}

// Meta describes the capture conditions a noise report was taken under.
type Meta struct {
	Camera     string
	Width      int
	Height     int
	FPS        float64
	Format     string
	AutoLocked bool
	Controls   string
}

// Close releases all Mats held by the report.
func (r *Report) Close() {
	r.Mean.Close()
	r.StdDev.Close()
	r.DeadMask.Close()
	r.NoisyMask.Close()
}

// Collector accumulates grayscale frames and computes per-pixel statistics
// (mean and standard deviation over time) to detect dead/stuck and noisy pixels.
type Collector struct {
	active bool

	target    int
	collected int
	started   time.Time

	sum   gocv.Mat
	sumSq gocv.Mat

	deadThreshold   float64 // stddev below this = "dead/stuck" pixel
	noisyMultiplier float64 // stddev above (avgStdDev * multiplier) = "noisy" pixel
}

// NewCollector creates a noise-test collector that accumulates `target` frames.
func NewCollector(target int, deadThreshold, noisyMultiplier float64) *Collector {
	return &Collector{
		target:          target,
		deadThreshold:   deadThreshold,
		noisyMultiplier: noisyMultiplier,
	}
}

// Active reports whether a scan is currently in progress.
func (c *Collector) Active() bool {
	return c.active
}

// Progress returns collected/target frame counts.
func (c *Collector) Progress() (int, int) {
	return c.collected, c.target
}

// Start resets the collector and begins a new scan for a frame of the given size.
func (c *Collector) Start(rows, cols int) {
	if c.active {
		return
	}

	// float64 accumulators keep precision on long runs (tens of thousands of frames)
	c.sum = gocv.NewMatWithSize(rows, cols, gocv.MatTypeCV64F)
	c.sumSq = gocv.NewMatWithSize(rows, cols, gocv.MatTypeCV64F)
	c.collected = 0
	c.started = time.Now()
	c.active = true
}

// Add feeds one grayscale (CV8UC1) frame into the accumulator.
// Returns true once `target` frames have been collected.
func (c *Collector) Add(gray gocv.Mat) bool {
	if !c.active {
		return false
	}

	f64 := gocv.NewMat()
	defer f64.Close()
	gray.ConvertTo(&f64, gocv.MatTypeCV64F)

	gocv.Accumulate(f64, &c.sum)
	gocv.AccumulateSquare(f64, &c.sumSq)

	c.collected++
	return c.collected >= c.target
}

// Finish computes the final report and releases accumulator resources.
// Must only be called after Add has returned true.
func (c *Collector) Finish() Report {
	defer func() {
		c.sum.Close()
		c.sumSq.Close()
		c.active = false
		c.collected = 0
	}()

	n := float64(c.collected)
	duration := time.Since(c.started)

	mean := gocv.NewMat()
	c.sum.ConvertToWithParams(&mean, gocv.MatTypeCV64F, float32(1.0/n), 0)

	meanSq := gocv.NewMat()
	defer meanSq.Close()
	gocv.Multiply(mean, mean, &meanSq)

	avgSq := gocv.NewMat()
	defer avgSq.Close()
	c.sumSq.ConvertToWithParams(&avgSq, gocv.MatTypeCV64F, float32(1.0/n), 0)

	variance := gocv.NewMat()
	defer variance.Close()
	gocv.Subtract(avgSq, meanSq, &variance)

	// rounding can leave tiny negative values for constant pixels; sqrt of them would be NaN
	gocv.Threshold(variance, &variance, 0, 0, gocv.ThresholdToZero)

	stddev := gocv.NewMat()
	gocv.Pow(variance, 0.5, &stddev)

	avgStdDev := stddev.Mean().Val1

	deadMask := gocv.NewMat()
	gocv.Threshold(stddev, &deadMask, float32(c.deadThreshold), 255, gocv.ThresholdBinaryInv)
	deadMask.ConvertTo(&deadMask, gocv.MatTypeCV8U)

	noisyMask := gocv.NewMat()
	gocv.Threshold(stddev, &noisyMask, float32(avgStdDev*c.noisyMultiplier), 255, gocv.ThresholdBinary)
	noisyMask.ConvertTo(&noisyMask, gocv.MatTypeCV8U)

	return Report{
		Mean:        mean,
		StdDev:      stddev,
		DeadMask:    deadMask,
		NoisyMask:   noisyMask,
		DeadCount:   gocv.CountNonZero(deadMask),
		NoisyCount:  gocv.CountNonZero(noisyMask),
		TotalPixels: mean.Rows() * mean.Cols(),
		MeanLevel:   mean.Mean().Val1,
		AvgStdDev:   avgStdDev,
		Frames:      c.collected,
		Duration:    duration,
	}
}

// SaveReport renders a visualization (dead pixels in blue, noisy pixels in red,
// overlaid on the average frame) and saves it to disk.
func SaveReport(r Report, meta Meta) error {
	if err := os.MkdirAll("noise_reports", 0755); err != nil {
		slog.Error("noise_reports dir error", "err", err)
		return err
	}

	meanU8 := gocv.NewMat()
	defer meanU8.Close()
	r.Mean.ConvertTo(&meanU8, gocv.MatTypeCV8U)

	visual := gocv.NewMat()
	defer visual.Close()
	_ = gocv.CvtColor(meanU8, &visual, gocv.ColorGrayToBGR)

	red := gocv.NewMatWithSizeFromScalar(gocv.NewScalar(0, 0, 255, 0), visual.Rows(), visual.Cols(), gocv.MatTypeCV8UC3)
	defer red.Close()
	red.CopyToWithMask(&visual, r.NoisyMask)

	blue := gocv.NewMatWithSizeFromScalar(gocv.NewScalar(255, 0, 0, 0), visual.Rows(), visual.Cols(), gocv.MatTypeCV8UC3)
	defer blue.Close()
	blue.CopyToWithMask(&visual, r.DeadMask)

	now := time.Now()
	stamp := now.Format("2006-01-02_15-04-05")
	path := filepath.Join("noise_reports", fmt.Sprintf("noise_report_%s.png", stamp))

	if !gocv.IMWrite(path, visual) {
		return fmt.Errorf("failed to save noise report")
	}

	deadPct := float64(r.DeadCount) / float64(r.TotalPixels) * 100
	noisyPct := float64(r.NoisyCount) / float64(r.TotalPixels) * 100

	report := fmt.Sprintf(
		"Noise test report - %s\n"+
			"Camera: %s\n"+
			"Resolution: %dx%d (%d pixels total)\n"+
			"FPS (driver): %.2f\n"+
			"Format: %s\n"+
			"Source: raw camera frame, grayscale (filters bypassed)\n"+
			"Auto controls locked: %s\n"+
			"Controls: %s\n"+
			"Frames: %d over %s\n"+
			"\n"+
			"Mean brightness:    %.3f\n"+
			"Avg temporal noise: %.4f (stddev, DN)\n"+
			"Dead/stuck pixels:  %d (%.4f%%)\n"+
			"Noisy pixels:       %d (%.4f%%)\n",
		now.Format("2006-01-02 15:04:05"),
		meta.Camera,
		r.Mean.Cols(), r.Mean.Rows(), r.TotalPixels,
		meta.FPS,
		meta.Format,
		yesNo(meta.AutoLocked),
		meta.Controls,
		r.Frames, r.Duration.Round(time.Millisecond),
		r.MeanLevel,
		r.AvgStdDev,
		r.DeadCount, deadPct,
		r.NoisyCount, noisyPct,
	)

	txtPath := filepath.Join("noise_reports", fmt.Sprintf("noise_report_%s.txt", stamp))
	if err := os.WriteFile(txtPath, []byte(report), 0644); err != nil {
		slog.Error("noise report txt save error", "err", err)
	}

	slog.Info("noise test complete",
		"dead_pixels", r.DeadCount,
		"noisy_pixels", r.NoisyCount,
		"total_pixels", r.TotalPixels,
		"avg_stddev", r.AvgStdDev,
		"auto_locked", meta.AutoLocked,
		"file", path,
	)

	return nil
}

// yesNo converts a boolean into "yes"/"no" for reports.
func yesNo(v bool) string {
	if v {
		return "yes"
	}
	return "no"
}

// Event represents noise-test lifecycle events.
type Event int

const (
	EventNone Event = iota
	EventStarted
	EventProgress
	EventDone
)

// Result describes the outcome of one UpdateNoiseTest call.
type Result struct {
	Event      Event
	Message    string
	DeadCount  int
	NoisyCount int
}

// UpdateNoiseTest drives the noise-test lifecycle: starts a scan when requested,
// feeds the current frame into the collector, and finalizes + saves the report
// once enough frames have been collected.
// frame must be the raw camera frame, not the filtered one.
func UpdateNoiseTest(c *Collector, state *media.State, frame gocv.Mat, meta Meta) Result {
	if state.NoiseTest && !c.Active() {
		c.Start(frame.Rows(), frame.Cols())
		state.NoiseTest = false

		msg := "Noise test started: keep camera still"
		if !meta.AutoLocked {
			msg += " (auto controls not locked, press L)"
		}
		return Result{Event: EventStarted, Message: msg}
	}

	if !c.Active() {
		return Result{Event: EventNone}
	}

	gray := gocv.NewMat()
	defer gray.Close()
	_ = gocv.CvtColor(frame, &gray, gocv.ColorBGRToGray)

	if c.Add(gray) {
		report := c.Finish()
		defer report.Close()

		if err := SaveReport(report, meta); err != nil {
			slog.Error("noise report save error", "err", err)
		}

		return Result{
			Event:      EventDone,
			Message:    fmt.Sprintf("Noise test done: noise=%.3f dead=%d noisy=%d", report.AvgStdDev, report.DeadCount, report.NoisyCount),
			DeadCount:  report.DeadCount,
			NoisyCount: report.NoisyCount,
		}
	}

	collected, target := c.Progress()
	return Result{
		Event:   EventProgress,
		Message: fmt.Sprintf("Analyzing noise... %d/%d", collected, target),
	}
}
