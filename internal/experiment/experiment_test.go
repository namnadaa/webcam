package experiment

import (
	"bufio"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"gocv.io/x/gocv"
)

func TestRunning(t *testing.T) {
	var r Running
	for _, v := range []float64{2, 4, 4, 4, 5, 5, 7, 9} {
		r.Add(v)
	}

	if r.N != 8 || r.Mean() != 5 || r.Min != 2 || r.Max != 9 {
		t.Errorf("Running = N %d mean %v min %v max %v", r.N, r.Mean(), r.Min, r.Max)
	}
	if want := math.Sqrt(32.0 / 7); math.Abs(r.StdDev()-want) > 1e-12 {
		t.Errorf("StdDev = %v, want %v", r.StdDev(), want)
	}
}

func TestTrend(t *testing.T) {
	var tr Trend
	for i := 0; i < 10; i++ {
		tr.Add(float64(i), 3+0.5*float64(i))
	}
	if math.Abs(tr.Slope()-0.5) > 1e-12 {
		t.Errorf("Slope = %v, want 0.5", tr.Slope())
	}

	var empty Trend
	if empty.Slope() != 0 {
		t.Errorf("empty Slope = %v, want 0", empty.Slope())
	}
}

func TestConfig(t *testing.T) {
	c := DefaultConfig()

	c.AdjustThreshold(-100)
	if c.Threshold != ThresholdMin {
		t.Errorf("threshold below range = %v, want %v", c.Threshold, ThresholdMin)
	}

	c.AdjustThreshold(0.3)
	if c.Threshold != 0.4 {
		t.Errorf("threshold = %v, want 0.4", c.Threshold)
	}

	for range Durations {
		c.NextDuration()
	}
	if c.DurationIndex != DefaultConfig().DurationIndex {
		t.Errorf("duration did not cycle back: %d", c.DurationIndex)
	}

	if ModeStability.Next() != ModeDark {
		t.Error("mode did not cycle back to dark")
	}
}

func TestFormatDuration(t *testing.T) {
	tests := map[time.Duration]string{
		30 * time.Second: "30s",
		5 * time.Minute:  "5m",
		90 * time.Second: "1m30s",
	}
	for d, want := range tests {
		if got := FormatDuration(d); got != want {
			t.Errorf("FormatDuration(%v) = %q, want %q", d, got, want)
		}
	}

	if got := FormatClock(125 * time.Second); got != "02:05" {
		t.Errorf("FormatClock = %q, want 02:05", got)
	}
}

func TestMeter(t *testing.T) {
	m := NewMeter()
	defer m.Close()

	first := frameOf(10)
	defer first.Close()
	s := m.Measure(first, time.Now())
	if s.HasNoise || s.Mean != 10 || s.SpatialStd != 0 {
		t.Errorf("first sample = %+v", s)
	}

	// half of the pixels change by 2 DN: diff stddev is 1, noise is 1/sqrt(2)
	second := frameOf(10)
	defer second.Close()
	half := second.RowRange(0, 2)
	half.SetTo(gocv.NewScalar(12, 12, 12, 0))
	half.Close()

	s = m.Measure(second, time.Now())
	if !s.HasNoise || math.Abs(s.Noise-1/math.Sqrt2) > 1e-9 {
		t.Errorf("Noise = %v, want %v", s.Noise, 1/math.Sqrt2)
	}
}

func TestSessionWritesResults(t *testing.T) {
	t.Chdir(t.TempDir())

	cfg := DefaultConfig()
	cfg.Threshold = 0.5
	meta := Meta{Camera: "test", Width: 16, Height: 4, FPS: 30, Format: "MJPG"}

	s, err := Start(cfg, meta, meta.Height, meta.Width)
	if err != nil {
		t.Fatal(err)
	}

	m := NewMeter()
	defer m.Close()

	now := time.Now()
	for i, v := range []float64{10, 12, 10, 12} {
		frame := frameOf(v)
		sample := m.Measure(frame, now.Add(time.Duration(i)*33*time.Millisecond))
		if done := s.Add(sample, m.Gray()); done {
			t.Fatal("session finished before its duration")
		}
		frame.Close()
	}

	summary, err := s.Finish(true)
	if err != nil {
		t.Fatal(err)
	}

	if summary.Frames != 4 || summary.AvgNoise != 0 || summary.Drift != 2 {
		t.Errorf("summary = %+v", summary)
	}

	for _, name := range []string{"meta.txt", "data.csv", "report.txt", "mean.png", "noise_map.png", "defects.png"} {
		if _, err := os.Stat(filepath.Join(summary.Dir, name)); err != nil {
			t.Errorf("missing %s: %v", name, err)
		}
	}

	if lines := readLines(t, filepath.Join(summary.Dir, "data.csv")); len(lines) != 5 {
		t.Errorf("data.csv has %d lines, want header + 4", len(lines))
	}

	report, _ := os.ReadFile(filepath.Join(summary.Dir, "report.txt"))
	for _, want := range []string{"stopped by user", "Pixel noise:      avg 1.0000", "NOT locked", "ZONES 8x6", "effective 30.30 FPS"} {
		if !strings.Contains(string(report), want) {
			t.Errorf("report.txt does not contain %q", want)
		}
	}
}

// frameOf returns a 4x16 BGR frame filled with one brightness value.
func frameOf(v float64) gocv.Mat {
	return gocv.NewMatWithSizeFromScalar(gocv.NewScalar(v, v, v, 0), 4, 16, gocv.MatTypeCV8UC3)
}

func readLines(t *testing.T, path string) []string {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	var lines []string
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		lines = append(lines, sc.Text())
	}
	return lines
}
