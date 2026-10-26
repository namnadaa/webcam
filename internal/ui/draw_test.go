package ui

import (
	"reflect"
	"testing"
)

func TestScaleFor(t *testing.T) {
	tests := map[int]float64{
		720:  1,
		1080: 1.5,
		480:  480.0 / 720,
		240:  minScale, // clamped
	}
	for rows, want := range tests {
		if got := scaleFor(rows); got != want {
			t.Errorf("scaleFor(%d) = %v, want %v", rows, got, want)
		}
	}
}

func TestSplitMenu(t *testing.T) {
	lines := []string{"a", "b", "", "c", "d", "e", "", "f", "g"}

	if cols := splitMenu(lines, 9); len(cols) != 1 {
		t.Errorf("menu that fits was split into %d columns", len(cols))
	}

	// mid is 5; the spacer at 6 is the closest one, the spacer itself is dropped
	want := [][]string{{"a", "b", "", "c", "d", "e"}, {"f", "g"}}
	if got := splitMenu(lines, 6); !reflect.DeepEqual(got, want) {
		t.Errorf("splitMenu(6) = %q, want %q", got, want)
	}

	// with 5 lines per column the spacer at 6 leaves a 6-line first column, so split in the middle
	want = [][]string{{"a", "b", "", "c", "d"}, {"e", "", "f", "g"}}
	if got := splitMenu(lines, 5); !reflect.DeepEqual(got, want) {
		t.Errorf("splitMenu(5) = %q, want %q", got, want)
	}
}

// menuColumns lays the menu out for a frame height the same way drawMenu does.
func menuColumns(rows int) ([][]string, int) {
	s := scaleFor(rows)
	line := int(lineHeight*s + 0.5)
	top := int(topPadding*s + 0.5)
	available := (rows - top - bottomLines*line) / line
	return splitMenu(menuLines(), available), available
}

func TestMenuColumns(t *testing.T) {
	// above the minimum scale lines shrink with the frame, so the menu keeps one column
	for _, rows := range []int{480, 720, 1080} {
		if cols, _ := menuColumns(rows); len(cols) != 1 {
			t.Errorf("menu at %dp uses %d columns, want 1", rows, len(cols))
		}
	}

	// at 240p the scale is clamped, the menu must be split and each column must fit
	cols, available := menuColumns(240)
	if len(cols) != 2 {
		t.Fatalf("menu at 240p uses %d columns, want 2", len(cols))
	}
	for i, c := range cols {
		if len(c) > available {
			t.Errorf("column %d has %d lines, only %d fit", i, len(c), available)
		}
	}
}
