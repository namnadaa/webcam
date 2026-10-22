package diagnostics

import (
	"math"
	"testing"

	"gocv.io/x/gocv"
)

func feed(t *testing.T, c *Collector, values ...float64) {
	t.Helper()
	for _, v := range values {
		frame := gocv.NewMatWithSizeFromScalar(gocv.NewScalar(v, 0, 0, 0), 4, 6, gocv.MatTypeCV8UC1)
		c.Add(frame)
		frame.Close()
	}
}

func TestCollector_AlternatingFrames(t *testing.T) {
	c := NewCollector(4, 1.0, 3.0)
	c.Start(4, 6)
	feed(t, c, 10, 20, 10, 20)

	r := c.Finish()
	defer r.Close()

	if math.Abs(r.MeanLevel-15) > 1e-9 {
		t.Errorf("MeanLevel = %v, want 15", r.MeanLevel)
	}
	if math.Abs(r.AvgStdDev-5) > 1e-9 {
		t.Errorf("AvgStdDev = %v, want 5", r.AvgStdDev)
	}
	if r.DeadCount != 0 {
		t.Errorf("DeadCount = %d, want 0", r.DeadCount)
	}
	if r.Frames != 4 || r.TotalPixels != 24 {
		t.Errorf("Frames=%d TotalPixels=%d, want 4 and 24", r.Frames, r.TotalPixels)
	}
}

func TestCollector_ConstantFramesHaveNoNaN(t *testing.T) {
	c := NewCollector(3, 1.0, 3.0)
	c.Start(4, 6)
	feed(t, c, 200, 200, 200)

	r := c.Finish()
	defer r.Close()

	if math.IsNaN(r.AvgStdDev) || r.AvgStdDev != 0 {
		t.Errorf("AvgStdDev = %v, want 0", r.AvgStdDev)
	}
	if r.DeadCount != r.TotalPixels {
		t.Errorf("DeadCount = %d, want all %d pixels", r.DeadCount, r.TotalPixels)
	}
}
