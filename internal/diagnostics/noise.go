package diagnostics

import (
	"fmt"
	"time"

	"gocv.io/x/gocv"
)

// Report holds the result of a per-pixel noise / dead-pixel scan.
type Report struct {
	Mean        gocv.Mat // average brightness per pixel (CV64F)
	StdDev      gocv.Mat // temporal standard deviation per pixel (CV64F)
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

// NewCollector creates a collector that accumulates `target` frames.
// target <= 0 means no limit: the caller decides when to call Finish.
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

// Add feeds one single-channel frame into the accumulator.
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
	return c.target > 0 && c.collected >= c.target
}

// Discard stops a scan without computing a report and releases accumulator resources.
func (c *Collector) Discard() {
	if !c.active {
		return
	}
	c.sum.Close()
	c.sumSq.Close()
	c.active = false
	c.collected = 0
}

// Finish computes the final report and releases accumulator resources.
// At least one frame must have been added.
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

// SaveDefectMap renders dead pixels in blue and noisy pixels in red over the average frame.
func SaveDefectMap(r Report, path string) error {
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

	if !gocv.IMWrite(path, visual) {
		return fmt.Errorf("failed to save defect map %s", path)
	}
	return nil
}

// SaveMeanMap saves the average frame as an 8-bit grayscale image.
func SaveMeanMap(r Report, path string) error {
	meanU8 := gocv.NewMat()
	defer meanU8.Close()
	r.Mean.ConvertTo(&meanU8, gocv.MatTypeCV8U)

	if !gocv.IMWrite(path, meanU8) {
		return fmt.Errorf("failed to save mean map %s", path)
	}
	return nil
}

// SaveNoiseMap saves the per-pixel temporal noise as a false-color heat map.
// Values from 0 to maxStdDev are mapped onto the full color range; brighter means noisier.
func SaveNoiseMap(r Report, path string, maxStdDev float64) error {
	if maxStdDev <= 0 {
		maxStdDev = 1
	}

	scaled := gocv.NewMat()
	defer scaled.Close()
	r.StdDev.ConvertToWithParams(&scaled, gocv.MatTypeCV8U, float32(255/maxStdDev), 0)

	heat := gocv.NewMat()
	defer heat.Close()
	gocv.ApplyColorMap(scaled, &heat, gocv.ColormapHot)

	if !gocv.IMWrite(path, heat) {
		return fmt.Errorf("failed to save noise map %s", path)
	}
	return nil
}
