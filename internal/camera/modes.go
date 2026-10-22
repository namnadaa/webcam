package camera

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// confirmTolerance is how far the measured FPS may be from the requested one
// for the frame rate to count as supported.
const confirmTolerance = 0.2

// FPSResult is one requested frame rate of a mode and what the camera really delivered.
type FPSResult struct {
	Requested float64 // FPS asked for
	Declared  float64 // FPS the driver reports after the request (often just echoes it)
	Measured  float64 // frames per second actually received; 0 if no frames came
	Resized   bool    // the driver changed the frame size to deliver this FPS
}

// Confirmed reports whether the measured frame rate matches the requested one.
func (r FPSResult) Confirmed() bool {
	if r.Measured <= 0 || r.Requested <= 0 || r.Resized {
		return false
	}
	return math.Abs(r.Measured-r.Requested)/r.Requested <= confirmTolerance
}

// Mode describes a resolution and pixel format the camera accepts together with its frame rates.
type Mode struct {
	Width        int
	Height       int
	Format       string      // pixel format to request, "" = driver default
	ActualFormat string      // pixel format the camera used while probing
	FPS          []float64   // selectable frame rates: confirmed ones, or presets when not measured
	Results      []FPSResult // all measurements; empty when the modes were not measured
}

// String returns the mode as "WxH" or "WxH MJPG" when a format is requested.
func (m Mode) String() string {
	if m.Format != "" {
		return fmt.Sprintf("%dx%d %s", m.Width, m.Height, m.Format)
	}
	return fmt.Sprintf("%dx%d", m.Width, m.Height)
}

// Measured reports whether the frame rates of the mode were measured.
func (m Mode) Measured() bool {
	return len(m.Results) > 0
}

// confirmedFPS returns the requested frame rates that were confirmed by measurement.
func confirmedFPS(results []FPSResult) []float64 {
	var fps []float64
	for _, r := range results {
		if r.Confirmed() {
			fps = append(fps, r.Requested)
		}
	}
	return fps
}

// PresetModes builds unmeasured modes from the predefined resolutions and FPS presets.
// Used when probing is skipped or fails.
func PresetModes() []Mode {
	modes := make([]Mode, 0, len(Resolutions))
	for _, r := range Resolutions {
		fps := make([]float64, len(FPSPresets))
		copy(fps, FPSPresets)
		modes = append(modes, Mode{Width: r.Width, Height: r.Height, FPS: fps})
	}
	return modes
}

// FindMode returns indices of the mode and frame rate closest to the given settings.
// A mode with the same pixel format is preferred over one with the same size only.
func FindMode(modes []Mode, w, h int, fps float64, format string) (int, int) {
	if len(modes) == 0 {
		return 0, 0
	}

	resIdx := -1
	for i, m := range modes {
		if m.Width != w || m.Height != h {
			continue
		}
		if m.Format == format {
			resIdx = i
			break
		}
		if resIdx < 0 {
			resIdx = i
		}
	}
	if resIdx < 0 {
		resIdx = 0
	}

	return resIdx, ClosestFPS(modes[resIdx].FPS, fps)
}

// ClosestFPS returns the index of the frame rate nearest to fps.
func ClosestFPS(list []float64, fps float64) int {
	best := 0
	for i, v := range list {
		if math.Abs(v-fps) < math.Abs(list[best]-fps) {
			best = i
		}
	}
	return best
}

// FormatFPS joins frame rates into a readable list, e.g. "15, 29.97, 30".
func FormatFPS(list []float64) string {
	if len(list) == 0 {
		return "none"
	}

	parts := make([]string, len(list))
	for i, v := range list {
		parts[i] = fmt.Sprintf("%g", v)
	}
	return strings.Join(parts, ", ")
}

// formatLabel describes the pixel format of a mode for the table.
func formatLabel(m Mode) string {
	known := m.ActualFormat != "" && m.ActualFormat != "unknown"

	switch {
	case m.Format == "" && known:
		return "auto (" + m.ActualFormat + ")"
	case m.Format == "":
		return "auto"
	case !known:
		return m.Format + " (unverified)" // the backend does not report the format
	case m.ActualFormat != m.Format:
		return m.Format + " -> " + m.ActualFormat
	}
	return m.Format
}

// PrintModes prints the numbered list of modes with requested, measured and driver-declared FPS.
func PrintModes(modes []Mode) {
	if len(modes) == 0 || !modes[0].Measured() {
		fmt.Println("\nCamera modes (NOT measured, presets: the camera may not support all of them)")
		fmt.Println("===============================================================")
		for i, m := range modes {
			fmt.Printf("[%d] %-12s FPS: %s\n", i+1, m.String(), FormatFPS(m.FPS))
		}
		fmt.Println("===============================================================")
		return
	}

	fmt.Println("\nCamera modes (measured)")
	fmt.Println("============================================================================")
	fmt.Printf("%-5s %-11s %-17s %10s %9s %8s  %s\n", "#", "Resolution", "Format", "Requested", "Measured", "Driver", "Status")
	fmt.Println("----------------------------------------------------------------------------")

	for i, m := range modes {
		for j, line := range resultLines(m.Results) {
			num, res, format := "", "", ""
			if j == 0 {
				num, res, format = fmt.Sprintf("[%d]", i+1), fmt.Sprintf("%dx%d", m.Width, m.Height), formatLabel(m)
			}
			fmt.Printf("%-5s %-11s %-17s %s\n", num, res, format, line)
		}
	}

	fmt.Println("============================================================================")
	fmt.Printf("Measured = frames really received. Not confirmed = more than %.0f%% off the request;\n", confirmTolerance*100)
	fmt.Println("such frame rates are not offered for selection.")
}

// resultLines formats measurements one per line; consecutive unconfirmed results are merged into one line.
func resultLines(results []FPSResult) []string {
	var lines []string

	for i := 0; i < len(results); {
		r := results[i]
		if r.Confirmed() {
			lines = append(lines, fmt.Sprintf("%10g %9.1f %8g  ok", r.Requested, r.Measured, r.Declared))
			i++
			continue
		}

		j := i
		for j < len(results) && !results[j].Confirmed() {
			j++
		}
		lines = append(lines, unconfirmedLine(results[i:j]))
		i = j
	}

	return lines
}

// unconfirmedLine merges a run of unconfirmed results: "60-240  30.0  60-240  not confirmed".
func unconfirmedLine(run []FPSResult) string {
	first, last := run[0], run[len(run)-1]

	requested := fmt.Sprintf("%g", first.Requested)
	if len(run) > 1 {
		requested = fmt.Sprintf("%g-%g", first.Requested, last.Requested)
	}

	lo, hi, sum := run[0].Measured, run[0].Measured, 0.0
	declLo, declHi := run[0].Declared, run[0].Declared
	resized := false
	for _, r := range run {
		lo, hi = math.Min(lo, r.Measured), math.Max(hi, r.Measured)
		declLo, declHi = math.Min(declLo, r.Declared), math.Max(declHi, r.Declared)
		sum += r.Measured
		resized = resized || r.Resized
	}

	// nearly equal values are shown as their average, a wider spread as a range
	measured := fmt.Sprintf("%.1f", sum/float64(len(run)))
	if hi-lo >= 0.5 {
		measured = fmt.Sprintf("%.1f-%.1f", lo, hi)
	}

	declared := fmt.Sprintf("%g", declLo)
	if declHi != declLo {
		declared = fmt.Sprintf("%g-%g", declLo, declHi)
	}

	status := "not confirmed"
	if resized {
		status += " (size changed)"
	}
	return fmt.Sprintf("%10s %9s %8s  %s", requested, measured, declared, status)
}

// AskMatrix asks which modes, each with all its frame rates, the mode matrix should compare.
// Returns nil when the user skips the matrix.
func AskMatrix(modes []Mode) []Mode {
	var input string

	for {
		fmt.Print("\nMode matrix: modes to compare (e.g. 1,3,5 or 'all', 'n' to skip): ")
		fmt.Scan(&input)

		input = strings.ToLower(strings.TrimSpace(input))
		if input == "n" {
			return nil
		}

		idx, err := ParseSelection(input, len(modes))
		if err != nil {
			fmt.Println(err)
			continue
		}

		selected := make([]Mode, 0, len(idx))
		runs := 0
		for _, i := range idx {
			selected = append(selected, modes[i])
			runs += max(1, len(modes[i].FPS))
		}
		fmt.Printf("Mode matrix: %d camera modes, start it in the camera window with [U]\n", runs)
		return selected
	}
}

// ParseSelection parses "all" or a comma-separated list of 1-based numbers up to n
// into unique 0-based indices in the given order.
func ParseSelection(input string, n int) ([]int, error) {
	if input == "all" {
		idx := make([]int, n)
		for i := range idx {
			idx[i] = i
		}
		return idx, nil
	}

	var idx []int
	seen := map[int]bool{}

	for _, part := range strings.Split(input, ",") {
		num, err := strconv.Atoi(strings.TrimSpace(part))
		if err != nil || num < 1 || num > n {
			return nil, fmt.Errorf("enter numbers from 1 to %d separated by commas, 'all' or 'n'", n)
		}
		if !seen[num] {
			seen[num] = true
			idx = append(idx, num-1)
		}
	}

	return idx, nil
}
