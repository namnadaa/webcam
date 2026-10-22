package experiment

import (
	"image"
	"webcam/internal/diagnostics"

	"gocv.io/x/gocv"
)

// Zone grid used to describe how level and noise vary across the field.
const (
	ZoneCols = 8
	ZoneRows = 6
)

// saturationLevel is the average brightness above which a pixel is treated as saturated.
const saturationLevel = 250

// Zone holds the statistics of one grid cell.
type Zone struct {
	Mean  float64 // average level, DN
	Noise float64 // average temporal noise, DN
}

// Field describes spatial behaviour of the sensor over the whole run.
type Field struct {
	MeanLevel     float64 // average level of the frame, DN
	NonUniformity float64 // stddev of per-pixel averages: DSNU in the dark, PRNU in light, DN
	RowFPN        float64 // stddev of row averages, DN
	ColFPN        float64 // stddev of column averages, DN
	AvgNoise      float64 // average per-pixel temporal noise, DN
	MaxNoise      float64 // noisiest pixel, DN
	AboveThresh   int     // pixels whose temporal noise exceeds the threshold
	Saturated     int     // pixels with average level >= saturationLevel
	Zones         [ZoneRows][ZoneCols]Zone
}

// analyzeField computes spatial statistics from the per-pixel mean and noise maps.
func analyzeField(r diagnostics.Report, threshold float64) Field {
	f := Field{
		MeanLevel: r.MeanLevel,
		AvgNoise:  r.AvgStdDev,
	}

	_, f.NonUniformity = meanStdDev(r.Mean)
	_, f.RowFPN = reducedStdDev(r.Mean, 1)
	_, f.ColFPN = reducedStdDev(r.Mean, 0)

	_, maxNoise, _, _ := gocv.MinMaxLoc(r.StdDev)
	f.MaxNoise = float64(maxNoise)

	f.AboveThresh = countAbove(r.StdDev, threshold)
	f.Saturated = countAbove(r.Mean, saturationLevel-0.5)

	w, h := r.Mean.Cols(), r.Mean.Rows()
	for zy := 0; zy < ZoneRows; zy++ {
		for zx := 0; zx < ZoneCols; zx++ {
			rect := image.Rect(zx*w/ZoneCols, zy*h/ZoneRows, (zx+1)*w/ZoneCols, (zy+1)*h/ZoneRows)
			if rect.Empty() {
				continue
			}

			meanRegion := r.Mean.Region(rect)
			noiseRegion := r.StdDev.Region(rect)
			f.Zones[zy][zx] = Zone{
				Mean:  meanRegion.Mean().Val1,
				Noise: noiseRegion.Mean().Val1,
			}
			meanRegion.Close()
			noiseRegion.Close()
		}
	}

	return f
}

// ZoneSpread returns the min and max zone averages, a coarse measure of field non-uniformity.
func (f Field) ZoneSpread() (minMean, maxMean float64) {
	minMean, maxMean = f.Zones[0][0].Mean, f.Zones[0][0].Mean
	for _, row := range f.Zones {
		for _, z := range row {
			minMean = min(minMean, z.Mean)
			maxMean = max(maxMean, z.Mean)
		}
	}
	return minMean, maxMean
}

// reducedStdDev averages a map along dim (1 = per row, 0 = per column) and returns the spread of those averages.
func reducedStdDev(src gocv.Mat, dim int) (float64, float64) {
	reduced := gocv.NewMat()
	defer reduced.Close()

	if err := gocv.Reduce(src, &reduced, dim, gocv.ReduceAvg, gocv.MatTypeCV64F); err != nil {
		return 0, 0
	}
	return meanStdDev(reduced)
}

// countAbove counts elements of a single-channel Mat strictly greater than level.
func countAbove(src gocv.Mat, level float64) int {
	mask := gocv.NewMat()
	defer mask.Close()

	gocv.Threshold(src, &mask, float32(level), 255, gocv.ThresholdBinary)
	mask.ConvertTo(&mask, gocv.MatTypeCV8U)
	return gocv.CountNonZero(mask)
}
