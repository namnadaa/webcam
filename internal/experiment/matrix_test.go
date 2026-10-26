package experiment

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMatrix(t *testing.T) {
	t.Chdir(t.TempDir())

	steps := []Step{{640, 480, 30, ""}, {1280, 720, 10, ""}, {1920, 1080, 5, "MJPG"}}
	m := NewMatrix(steps, DefaultConfig())

	if m.Progress() != "1/3" || m.Current().String() != "640x480@30" {
		t.Fatalf("start: %s %s", m.Progress(), m.Current())
	}

	if !m.Record(Summary{Width: 640, Height: 480, FPS: 30, AvgNoise: 1.25, Dir: "a"}) {
		t.Fatal("Record returned false with steps left")
	}
	if m.Current() != steps[1] {
		t.Errorf("current = %s, want %s", m.Current(), steps[1])
	}
	m.Record(Summary{Width: 1280, Height: 720, FPS: 10, AvgNoise: 0.75, Dir: "b"})

	path, err := m.Save(true)
	if err != nil {
		t.Fatal(err)
	}

	txt, _ := os.ReadFile(path)
	for _, want := range []string{"2 of 3 measured (aborted by user)", "640x480@30", "1280x720@10", "1.2500", "0.7500"} {
		if !strings.Contains(string(txt), want) {
			t.Errorf("summary.txt does not contain %q", want)
		}
	}

	csv, _ := os.ReadFile(filepath.Join(filepath.Dir(path), "summary.csv"))
	if lines := strings.Count(string(csv), "\n"); lines != 3 {
		t.Errorf("summary.csv has %d lines, want header + 2", lines)
	}
}
