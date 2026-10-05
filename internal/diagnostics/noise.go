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
	Mean        gocv.Mat // average brightness per pixel (CV32F)
	StdDev      gocv.Mat // standard deviation per pixel (CV32F)
	DeadMask    gocv.Mat // 255 where a pixel never changes (stuck/dead)
	NoisyMask   gocv.Mat // 255 where a pixel is an outlier by noise level
	DeadCount   int
	NoisyCount  int
	TotalPixels int
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

	c.sum = gocv.NewMatWithSize(rows, cols, gocv.MatTypeCV32F)
	c.sumSq = gocv.NewMatWithSize(rows, cols, gocv.MatTypeCV32F)
	c.collected = 0
	c.active = true
}

// Add feeds one grayscale (CV8UC1) frame into the accumulator.
// Returns true once `target` frames have been collected.
func (c *Collector) Add(gray gocv.Mat) bool {
	if !c.active {
		return false
	}

	f32 := gocv.NewMat()
	defer f32.Close()
	gray.ConvertTo(&f32, gocv.MatTypeCV32F)

	gocv.Accumulate(f32, &c.sum)
	gocv.AccumulateSquare(f32, &c.sumSq)

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

	mean := gocv.NewMat()
	c.sum.ConvertToWithParams(&mean, gocv.MatTypeCV32F, float32(1.0/n), 0)

	meanSq := gocv.NewMat()
	defer meanSq.Close()
	gocv.Multiply(mean, mean, &meanSq)

	avgSq := gocv.NewMat()
	defer avgSq.Close()
	c.sumSq.ConvertToWithParams(&avgSq, gocv.MatTypeCV32F, float32(1.0/n), 0)

	variance := gocv.NewMat()
	defer variance.Close()
	gocv.Subtract(avgSq, meanSq, &variance)

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
	}
}

// SaveReport renders a visualization (dead pixels in blue, noisy pixels in red,
// overlaid on the average frame) and saves it to disk.
func SaveReport(r Report) error {
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

	filename := fmt.Sprintf("noise_report_%s.png", time.Now().Format("2006-01-02_15-04-05"))
	path := filepath.Join("noise_reports", filename)

	if !gocv.IMWrite(path, visual) {
		return fmt.Errorf("failed to save noise report")
	}

	deadPct := float64(r.DeadCount) / float64(r.TotalPixels) * 100
	noisyPct := float64(r.NoisyCount) / float64(r.TotalPixels) * 100

	report := fmt.Sprintf(
		"Noise test report - %s\n"+
			"Resolution: %dx%d (%d pixels total)\n"+
			"Dead/stuck pixels:  %d (%.4f%%)\n"+
			"Noisy pixels:       %d (%.4f%%)\n",
		time.Now().Format("2006-01-02 15:04:05"),
		r.Mean.Cols(), r.Mean.Rows(), r.TotalPixels,
		r.DeadCount, deadPct,
		r.NoisyCount, noisyPct,
	)

	txtPath := filepath.Join("noise_reports", fmt.Sprintf("noise_report_%s.txt", time.Now().Format("2006-01-02_15-04-05")))
	if err := os.WriteFile(txtPath, []byte(report), 0644); err != nil {
		slog.Error("noise report txt save error", "err", err)
	}

	slog.Info("noise test complete",
		"dead_pixels", r.DeadCount,
		"noisy_pixels", r.NoisyCount,
		"total_pixels", r.TotalPixels,
		"file", path,
	)

	return nil
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
func UpdateNoiseTest(c *Collector, state *media.State, frame gocv.Mat) Result {
	if state.NoiseTest && !c.Active() {
		c.Start(frame.Rows(), frame.Cols())
		state.NoiseTest = false
		return Result{Event: EventStarted, Message: "Noise test started: keep camera still"}
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

		if err := SaveReport(report); err != nil {
			slog.Error("noise report save error", "err", err)
		}

		return Result{
			Event:      EventDone,
			Message:    fmt.Sprintf("Noise test done: dead=%d noisy=%d", report.DeadCount, report.NoisyCount),
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
