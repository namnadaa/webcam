package graph

import (
	"testing"
	"time"
)

func TestNiceCeil(t *testing.T) {
	tests := map[float64]float64{
		0:    1,
		0.73: 1,
		1.3:  2,
		2.4:  5,
		7:    10,
		13:   20,
	}
	for v, want := range tests {
		if got := niceCeil(v); got != want {
			t.Errorf("niceCeil(%v) = %v, want %v", v, got, want)
		}
	}
}

func TestScale(t *testing.T) {
	if y := scale(0, 0, 10, 40, 240); y != 240 {
		t.Errorf("bottom = %d, want 240", y)
	}
	if y := scale(10, 0, 10, 40, 240); y != 40 {
		t.Errorf("top = %d, want 40", y)
	}
	if y := scale(5, 0, 10, 40, 240); y != 140 {
		t.Errorf("middle = %d, want 140", y)
	}
	if y := scale(99, 0, 10, 40, 240); y != 40 {
		t.Errorf("clamped = %d, want 40", y)
	}
	if y := scale(1, 5, 5, 40, 240); y != 240 {
		t.Errorf("empty range = %d, want 240", y)
	}
}

func TestTrimAndRange(t *testing.T) {
	now := time.Now()
	points := []Point{
		{Time: now.Add(-90 * time.Second), Mean: 1},
		{Time: now.Add(-30 * time.Second), Mean: 5, Noise: 0.5, HasNoise: true},
		{Time: now, Mean: 3},
	}

	points = trim(points, now.Add(-Span))
	if len(points) != 2 {
		t.Fatalf("trim kept %d points, want 2", len(points))
	}

	lo, hi := rangeOf(points, meanOf)
	if lo != 3 || hi != 5 {
		t.Errorf("rangeOf = %v..%v, want 3..5", lo, hi)
	}
	if m := maxOf(points, noiseOf); m != 0.5 {
		t.Errorf("maxOf noise = %v, want 0.5", m)
	}
}
