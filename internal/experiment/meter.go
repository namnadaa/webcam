package experiment

import (
	"math"
	"time"

	"gocv.io/x/gocv"
)

// Sample holds the measurements of one frame.
type Sample struct {
	Time       time.Time
	Mean       float64 // average brightness of the frame, DN
	SpatialStd float64 // brightness spread across the frame (scene + fixed pattern + temporal noise), DN
	Noise      float64 // temporal noise: stddev(frame - previous frame) / sqrt(2), DN
	HasNoise   bool    // false for the first frame, when there is no previous frame yet
}

// Meter converts raw frames to grayscale and measures them frame by frame.
type Meter struct {
	gray gocv.Mat // current frame, CV8U
	cur  gocv.Mat // current frame, CV64F
	prev gocv.Mat // previous frame, CV64F
	diff gocv.Mat
	has  bool
}

// NewMeter allocates the working buffers.
func NewMeter() *Meter {
	return &Meter{
		gray: gocv.NewMat(),
		cur:  gocv.NewMat(),
		prev: gocv.NewMat(),
		diff: gocv.NewMat(),
	}
}

// Close releases the working buffers.
func (m *Meter) Close() {
	m.gray.Close()
	m.cur.Close()
	m.prev.Close()
	m.diff.Close()
}

// Reset forgets the previous frame, e.g. after the camera mode changed.
func (m *Meter) Reset() {
	m.has = false
}

// Measure analyzes a raw BGR (or grayscale) camera frame.
func (m *Meter) Measure(frame gocv.Mat, t time.Time) Sample {
	if frame.Channels() == 1 {
		frame.CopyTo(&m.gray)
	} else {
		_ = gocv.CvtColor(frame, &m.gray, gocv.ColorBGRToGray)
	}
	m.gray.ConvertTo(&m.cur, gocv.MatTypeCV64F)

	s := Sample{Time: t}
	s.Mean, s.SpatialStd = meanStdDev(m.cur)

	if m.has && m.prev.Rows() == m.cur.Rows() && m.prev.Cols() == m.cur.Cols() {
		gocv.Subtract(m.cur, m.prev, &m.diff)
		_, diffStd := meanStdDev(m.diff)
		s.Noise = diffStd / math.Sqrt2
		s.HasNoise = true
	}

	m.cur.CopyTo(&m.prev)
	m.has = true

	return s
}

// Gray returns the last measured frame in grayscale (CV8U). Valid until the next Measure call.
func (m *Meter) Gray() gocv.Mat {
	return m.gray
}

// meanStdDev returns the mean and standard deviation of a single-channel Mat.
func meanStdDev(src gocv.Mat) (float64, float64) {
	mean := gocv.NewMat()
	defer mean.Close()
	std := gocv.NewMat()
	defer std.Close()

	gocv.MeanStdDev(src, &mean, &std)
	return mean.GetDoubleAt(0, 0), std.GetDoubleAt(0, 0)
}
