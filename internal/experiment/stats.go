package experiment

import "math"

// Running accumulates count, mean, min, max and standard deviation of a series
// without storing it (Welford's algorithm, stable on long runs).
type Running struct {
	N    int
	mean float64
	m2   float64
	Min  float64
	Max  float64
}

// Add includes one value.
func (r *Running) Add(v float64) {
	r.N++
	if r.N == 1 {
		r.Min, r.Max = v, v
	} else {
		r.Min = math.Min(r.Min, v)
		r.Max = math.Max(r.Max, v)
	}

	d := v - r.mean
	r.mean += d / float64(r.N)
	r.m2 += d * (v - r.mean)
}

// Mean returns the average, or 0 for an empty series.
func (r *Running) Mean() float64 {
	return r.mean
}

// StdDev returns the sample standard deviation, or 0 for fewer than two values.
func (r *Running) StdDev() float64 {
	if r.N < 2 {
		return 0
	}
	return math.Sqrt(r.m2 / float64(r.N-1))
}

// Trend fits y = a + b*t by least squares without storing the points.
type Trend struct {
	n                float64
	st, sy, stt, sty float64
}

// Add includes one point.
func (tr *Trend) Add(t, y float64) {
	tr.n++
	tr.st += t
	tr.sy += y
	tr.stt += t * t
	tr.sty += t * y
}

// Slope returns b (units of y per unit of t), or 0 when it cannot be determined.
func (tr *Trend) Slope() float64 {
	den := tr.n*tr.stt - tr.st*tr.st
	if tr.n < 2 || den == 0 {
		return 0
	}
	return (tr.n*tr.sty - tr.st*tr.sy) / den
}
