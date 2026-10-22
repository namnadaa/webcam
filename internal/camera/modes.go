package camera

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"

	"gocv.io/x/gocv"
)

// Mode describes a resolution the camera accepts together with its frame rates.
type Mode struct {
	Width  int
	Height int
	FPS    []float64
}

// String returns the resolution as "WxH".
func (m Mode) String() string {
	return fmt.Sprintf("%dx%d", m.Width, m.Height)
}

// PresetModes builds modes from the predefined resolutions and FPS presets.
// Used as a fallback when the camera cannot be probed.
func PresetModes() []Mode {
	modes := make([]Mode, 0, len(Resolutions))
	for _, r := range Resolutions {
		fps := make([]float64, len(FPSPresets))
		copy(fps, FPSPresets)
		modes = append(modes, Mode{Width: r.Width, Height: r.Height, FPS: fps})
	}
	return modes
}

// ProbeModes asks the camera which of the preset resolutions and frame rates it actually applies.
// Every preset is requested, the applied value is read back and duplicates are dropped,
// so the result only contains modes the driver really switched to.
func ProbeModes(index int, api API) ([]Mode, error) {
	cam, err := Open(Config{Index: index, API: api})
	if err != nil {
		return nil, err
	}
	defer cam.Close()

	var modes []Mode

	defW, defH := cam.ActualSize()
	modes = addMode(modes, defW, defH, cam.ActualFPS())

	frame := gocv.NewMat()
	defer frame.Close()

	for _, r := range Resolutions {
		fmt.Printf("\rProbing %-10s", fmt.Sprintf("%dx%d", r.Width, r.Height))

		cam.Set(gocv.VideoCaptureFrameWidth, float64(r.Width))
		cam.Set(gocv.VideoCaptureFrameHeight, float64(r.Height))

		w, h, ok := readFrameSize(cam, &frame)
		if !ok {
			w, h = cam.ActualSize()
		}

		for _, fps := range FPSPresets {
			cam.Set(gocv.VideoCaptureFPS, fps)
			modes = addMode(modes, w, h, cam.ActualFPS())
		}
	}
	fmt.Printf("\r%-30s\r", "")

	return modes, nil
}

// readFrameSize grabs a frame so the driver applies pending settings and returns its size.
func readFrameSize(cam *Camera, frame *gocv.Mat) (int, int, bool) {
	const attempts = 5
	for i := 0; i < attempts; i++ {
		if cam.Read(frame) && !frame.Empty() {
			return frame.Cols(), frame.Rows(), true
		}
	}
	return 0, 0, false
}

// addMode inserts a resolution/FPS pair keeping modes and frame rates sorted and unique.
func addMode(modes []Mode, w, h int, fps float64) []Mode {
	if w <= 0 || h <= 0 {
		return modes
	}

	fps = math.Round(fps*100) / 100

	i := sort.Search(len(modes), func(i int) bool {
		return modes[i].Width*modes[i].Height > w*h ||
			(modes[i].Width*modes[i].Height == w*h && modes[i].Width >= w)
	})

	if i == len(modes) || modes[i].Width != w || modes[i].Height != h {
		modes = append(modes, Mode{})
		copy(modes[i+1:], modes[i:])
		modes[i] = Mode{Width: w, Height: h}
	}

	if fps <= 0 {
		return modes
	}

	m := &modes[i]
	j := sort.SearchFloat64s(m.FPS, fps)
	if j < len(m.FPS) && m.FPS[j] == fps {
		return modes
	}
	m.FPS = append(m.FPS, 0)
	copy(m.FPS[j+1:], m.FPS[j:])
	m.FPS[j] = fps

	return modes
}

// FindMode returns indices of the mode and frame rate closest to the given settings.
func FindMode(modes []Mode, w, h int, fps float64) (int, int) {
	if len(modes) == 0 {
		return 0, 0
	}

	resIdx := 0
	for i, m := range modes {
		if m.Width == w && m.Height == h {
			resIdx = i
			break
		}
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
		return "unknown"
	}

	parts := make([]string, len(list))
	for i, v := range list {
		parts[i] = fmt.Sprintf("%g", v)
	}
	return strings.Join(parts, ", ")
}

// PrintModes prints the numbered list of supported resolutions and frame rates.
func PrintModes(modes []Mode) {
	fmt.Println("\nSupported modes (applied by the driver)")
	fmt.Println("===============================================================")
	for i, m := range modes {
		fmt.Printf("[%d] %-12s FPS: %s\n", i+1, m.String(), FormatFPS(m.FPS))
	}
	fmt.Println("===============================================================")
}

// AskMatrix asks which resolutions, each with all its frame rates, the mode matrix should compare.
// Returns nil when the user skips the matrix.
func AskMatrix(modes []Mode) []Mode {
	var input string

	for {
		fmt.Print("\nMode matrix: resolutions to compare (e.g. 1,3,5 or 'all', 'n' to skip): ")
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
