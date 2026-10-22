package experiment

import (
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// squareWave returns levels for frames at 10 FPS: OFF at 20 DN, ON at 120 DN, switching every period/2.
func squareWave(frames, halfPeriod int) []float64 {
	levels := make([]float64, frames)
	for i := range levels {
		levels[i] = 20
		if (i/halfPeriod)%2 == 1 {
			levels[i] = 120
		}
		levels[i] += 0.3 * math.Sin(float64(i)) // small noise, far below the hysteresis band
	}
	return levels
}

func TestClassifier(t *testing.T) {
	var c Classifier
	start := time.Now()
	counts := map[LightState]int{}

	levels := squareWave(200, 10) // 1 s ON, 1 s OFF at 10 FPS
	for i, v := range levels {
		counts[c.Classify(v, start.Add(time.Duration(i)*100*time.Millisecond))]++
	}

	// the first OFF segment is unknown until the light turns on for the first time
	if counts[StateUnknown] != 10 {
		t.Errorf("unknown frames = %d, want 10", counts[StateUnknown])
	}
	if c.Switches != 18 {
		t.Errorf("switches = %d, want 18", c.Switches)
	}
	if counts[StateTransition] != 19*settleFrames {
		t.Errorf("transition frames = %d, want %d", counts[StateTransition], 19*settleFrames)
	}
	if math.Abs(c.Period()-2) > 1e-9 {
		t.Errorf("period = %v s, want 2", c.Period())
	}
}

func TestClassifierNoContrast(t *testing.T) {
	var c Classifier
	now := time.Now()
	for i := 0; i < 50; i++ {
		if s := c.Classify(50+float64(i%3), now); s != StateUnknown {
			t.Fatalf("frame %d: state %v, want unknown for a constant level", i, s)
		}
	}
}

func TestModulatorSession(t *testing.T) {
	t.Chdir(t.TempDir())

	cfg := DefaultConfig()
	cfg.Mode = ModeModulator
	meta := Meta{Camera: "test", Width: 16, Height: 4, FPS: 10, AutoLocked: true}

	s, err := Start(cfg, meta, meta.Height, meta.Width)
	if err != nil {
		t.Fatal(err)
	}

	m := NewMeter()
	defer m.Close()

	now := time.Now()
	for i, v := range squareWave(80, 10) {
		frame := frameOf(math.Round(v))
		s.Add(m.Measure(frame, now.Add(time.Duration(i)*100*time.Millisecond)), m.Gray())
		frame.Close()
	}

	summary, err := s.Finish(false)
	if err != nil {
		t.Fatal(err)
	}

	if summary.Level != 120 || summary.OffLevel != 20 || summary.Signal != 100 {
		t.Errorf("levels ON %v OFF %v signal %v, want 120 / 20 / 100", summary.Level, summary.OffLevel, summary.Signal)
	}
	if summary.Switches != 6 {
		t.Errorf("switches = %d, want 6", summary.Switches)
	}

	for _, name := range []string{"mean_on.png", "mean_off.png", "noise_map_on.png", "defects_off.png"} {
		if _, err := os.Stat(filepath.Join(summary.Dir, name)); err != nil {
			t.Errorf("missing %s: %v", name, err)
		}
	}

	report, _ := os.ReadFile(filepath.Join(summary.Dir, "report.txt"))
	for _, want := range []string{"MODULATION", "Signal:           100.000 DN", "LIGHT ON", "LIGHT OFF", "Period:           2.00 s"} {
		if !strings.Contains(string(report), want) {
			t.Errorf("report.txt does not contain %q", want)
		}
	}

	csv, _ := os.ReadFile(filepath.Join(summary.Dir, "data.csv"))
	for _, want := range []string{",on\n", ",off\n", ",transition\n", ",unknown\n"} {
		if !strings.Contains(string(csv), want) {
			t.Errorf("data.csv has no %q rows", strings.TrimSpace(want))
		}
	}
}
