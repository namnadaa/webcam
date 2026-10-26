package graph

import (
	"bufio"
	"fmt"
	"image"
	"image/color"
	"log/slog"
	"math"
	"os"
	"path/filepath"
	"time"
	"webcam/internal/helpers"
	"webcam/internal/i18n"
	"webcam/internal/textdraw"

	"gocv.io/x/gocv"
)

// WindowTitle is the name of the graph window.
const WindowTitle = "Noise graph"

// LogDir is where graph data is written while the window is open.
const LogDir = "noise_log"

const (
	canvasW = 900
	canvasH = 480

	plotLeft  = 70
	plotRight = canvasW - 20

	noiseTop    = 40
	noiseBottom = 260
	levelTop    = 300
	levelBottom = 450

	drawInterval  = 100 * time.Millisecond
	flushInterval = time.Second

	// Span is how much history the graph shows.
	Span = 60 * time.Second
)

var (
	colorBackground = color.RGBA{20, 20, 20, 0}
	colorGrid       = color.RGBA{60, 60, 60, 0}
	colorText       = color.RGBA{220, 220, 220, 0}
	colorNoise      = color.RGBA{0, 220, 0, 0}
	colorNoiseHigh  = color.RGBA{255, 60, 60, 0}
	colorThreshold  = color.RGBA{255, 60, 60, 0}
	colorLevel      = color.RGBA{0, 200, 255, 0}
)

// Point is one frame's measurements.
type Point struct {
	Time     time.Time
	Mean     float64
	Noise    float64
	HasNoise bool
}

// Graph shows the live noise and level in a separate window and logs them to CSV.
type Graph struct {
	win    *gocv.Window
	canvas gocv.Mat
	points []Point

	opened    time.Time
	lastDraw  time.Time
	lastFlush time.Time

	path string
	file *os.File
	csv  *bufio.Writer
}

// Open creates the graph window and the CSV log file.
func Open() (*Graph, error) {
	if err := os.MkdirAll(LogDir, 0755); err != nil {
		return nil, fmt.Errorf("create %s: %w", LogDir, err)
	}

	now := time.Now()
	path := filepath.Join(LogDir, fmt.Sprintf("noise_%s.csv", now.Format("2006-01-02_15-04-05")))

	file, err := os.Create(path)
	if err != nil {
		return nil, fmt.Errorf("create graph log: %w", err)
	}

	w := bufio.NewWriter(file)
	w.WriteString(helpers.CSVBOM)
	w.WriteString(helpers.CSVRow("time", "t_sec", "mean", "temporal_noise", "threshold", "above_threshold"))

	slog.Info("noise graph opened", "file", path)

	return &Graph{
		win:       gocv.NewWindow(WindowTitle),
		canvas:    gocv.NewMatWithSize(canvasH, canvasW, gocv.MatTypeCV8UC3),
		opened:    now,
		lastFlush: now,
		path:      path,
		file:      file,
		csv:       w,
	}, nil
}

// Path returns the CSV log file.
func (g *Graph) Path() string {
	return g.path
}

// Close closes the window and the CSV log.
func (g *Graph) Close() error {
	flushErr := g.csv.Flush()
	closeErr := g.file.Close()

	g.canvas.Close()
	_ = g.win.Close()

	slog.Info("noise graph closed", "file", g.path)

	if flushErr != nil {
		return flushErr
	}
	return closeErr
}

// Add records one frame and redraws the window at most every drawInterval.
func (g *Graph) Add(p Point, threshold float64) {
	g.points = append(g.points, p)
	g.points = trim(g.points, p.Time.Add(-Span))

	g.log(p, threshold)

	if p.Time.Sub(g.lastDraw) >= drawInterval {
		g.draw(p.Time, threshold)
		g.win.IMShow(g.canvas)
		g.lastDraw = p.Time
	}
}

// log appends a CSV row and flushes it to disk once a second.
func (g *Graph) log(p Point, threshold float64) {
	noise, above := "", "0"
	if p.HasNoise {
		noise = helpers.CSVFloat(p.Noise, 4)
		if p.Noise > threshold {
			above = "1"
		}
	}

	g.csv.WriteString(helpers.CSVRow(
		p.Time.Format("15:04:05.000"),
		helpers.CSVFloat(p.Time.Sub(g.opened).Seconds(), 3),
		helpers.CSVFloat(p.Mean, 4),
		noise,
		helpers.CSVFloat(threshold, 1),
		above,
	))

	if time.Since(g.lastFlush) >= flushInterval {
		if err := g.csv.Flush(); err != nil {
			slog.Error("noise graph csv flush", "err", err)
		}
		g.lastFlush = time.Now()
	}
}

// draw renders both plots onto the canvas.
func (g *Graph) draw(now time.Time, threshold float64) {
	g.canvas.SetTo(gocv.NewScalar(float64(colorBackground.B), float64(colorBackground.G), float64(colorBackground.R), 0))

	from := now.Add(-Span)
	last := g.points[len(g.points)-1]

	// ===== Temporal noise =====
	noiseMax := niceCeil(math.Max(maxOf(g.points, noiseOf), threshold) * 1.15)
	drawAxes(&g.canvas, noiseTop, noiseBottom, 0, noiseMax)

	thrY := scale(threshold, 0, noiseMax, noiseTop, noiseBottom)
	_ = gocv.Line(&g.canvas, image.Pt(plotLeft, thrY), image.Pt(plotRight, thrY), colorThreshold, 1)
	thrLabel := i18n.Tf("threshold %.1f", threshold)
	putText(&g.canvas, thrLabel, plotRight-textdraw.Width(thrLabel, textSize)-5, noiseTop-12, colorThreshold)

	noiseClr := colorNoise
	if last.HasNoise && last.Noise > threshold {
		noiseClr = colorNoiseHigh
	}
	drawSeries(&g.canvas, g.points, noiseOf, from, 0, noiseMax, noiseTop, noiseBottom, noiseClr)

	title := i18n.T("Temporal noise, DN: --")
	if last.HasNoise {
		title = i18n.Tf("Temporal noise, DN: %.3f", last.Noise)
	}
	putText(&g.canvas, title, plotLeft, noiseTop-12, noiseClr)

	// ===== Level =====
	lo, hi := rangeOf(g.points, meanOf)
	lo, hi = math.Floor(lo)-1, math.Ceil(hi)+1
	drawAxes(&g.canvas, levelTop, levelBottom, lo, hi)
	drawSeries(&g.canvas, g.points, meanOf, from, lo, hi, levelTop, levelBottom, colorLevel)
	putText(&g.canvas, i18n.Tf("Level, DN: %.2f", last.Mean), plotLeft, levelTop-12, colorLevel)

	// ===== Time axis =====
	for s := 0; s <= int(Span.Seconds()); s += 10 {
		x := plotLeft + (plotRight-plotLeft)*s/int(Span.Seconds())
		label := i18n.Tf("-%ds", int(Span.Seconds())-s)
		if s == int(Span.Seconds()) {
			label = i18n.T("0s")
		}
		putText(&g.canvas, label, x-15, levelBottom+20, colorText)
	}
}

// drawAxes draws the plot frame with three horizontal grid lines and their values.
func drawAxes(img *gocv.Mat, top, bottom int, lo, hi float64) {
	_ = gocv.Rectangle(img, image.Rect(plotLeft, top, plotRight, bottom), colorGrid, 1)

	for i := 0; i <= 2; i++ {
		v := lo + (hi-lo)*float64(i)/2
		y := scale(v, lo, hi, top, bottom)
		_ = gocv.Line(img, image.Pt(plotLeft, y), image.Pt(plotRight, y), colorGrid, 1)
		putText(img, fmt.Sprintf("%.2f", v), 8, y+5, colorText)
	}
}

// drawSeries draws one value of the points as a polyline over the last Span.
func drawSeries(img *gocv.Mat, points []Point, value func(Point) (float64, bool), from time.Time, lo, hi float64, top, bottom int, clr color.RGBA) {
	line := make([]image.Point, 0, len(points))
	for _, p := range points {
		v, ok := value(p)
		if !ok {
			continue
		}
		x := plotLeft + int(float64(plotRight-plotLeft)*p.Time.Sub(from).Seconds()/Span.Seconds())
		line = append(line, image.Pt(x, scale(v, lo, hi, top, bottom)))
	}

	if len(line) < 2 {
		return
	}

	pv := gocv.NewPointsVectorFromPoints([][]image.Point{line})
	defer pv.Close()
	_ = gocv.Polylines(img, pv, false, clr, 1)
}

// textSize is the font size of the graph labels, pixels.
const textSize = 14

// putText draws small plain text with its baseline at y.
func putText(img *gocv.Mat, text string, x, y int, clr color.RGBA) {
	textdraw.Draw(img, text, x, y, textSize, 0, clr)
}

func noiseOf(p Point) (float64, bool) { return p.Noise, p.HasNoise }
func meanOf(p Point) (float64, bool)  { return p.Mean, true }

// trim drops points older than from.
func trim(points []Point, from time.Time) []Point {
	i := 0
	for i < len(points) && points[i].Time.Before(from) {
		i++
	}
	return points[i:]
}

// scale maps v from [lo, hi] onto pixel rows [bottom, top], clamping to the plot.
func scale(v, lo, hi float64, top, bottom int) int {
	if hi <= lo {
		return bottom
	}
	y := float64(bottom) - (v-lo)/(hi-lo)*float64(bottom-top)
	return int(math.Round(math.Min(float64(bottom), math.Max(float64(top), y))))
}

// maxOf returns the largest available value, or 0.
func maxOf(points []Point, value func(Point) (float64, bool)) float64 {
	m := 0.0
	for _, p := range points {
		if v, ok := value(p); ok {
			m = math.Max(m, v)
		}
	}
	return m
}

// rangeOf returns the smallest and largest available values.
func rangeOf(points []Point, value func(Point) (float64, bool)) (float64, float64) {
	lo, hi := math.Inf(1), math.Inf(-1)
	for _, p := range points {
		if v, ok := value(p); ok {
			lo, hi = math.Min(lo, v), math.Max(hi, v)
		}
	}
	if lo > hi {
		return 0, 0
	}
	return lo, hi
}

// niceCeil rounds v up to 1, 2 or 5 times a power of ten, so axis values stay readable.
func niceCeil(v float64) float64 {
	if v <= 0 {
		return 1
	}
	exp := math.Pow(10, math.Floor(math.Log10(v)))
	for _, m := range []float64{1, 2, 5, 10} {
		if v <= m*exp {
			return m * exp
		}
	}
	return 10 * exp
}
