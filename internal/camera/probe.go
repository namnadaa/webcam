package camera

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"gocv.io/x/gocv"
)

// Probe timing: after every change a few frames are dropped while the driver switches,
// then frames are timed until enough are collected or the time is up.
const (
	skipFrames     = 2
	measureFrames  = 12
	measureTimeout = 1500 * time.Millisecond
)

// Formats tried in addition to the driver default, and the resolution they are tried at.
var (
	formatProbes          = []string{"MJPG", "YUY2"}
	formatProbeResolution = Resolution{1280, 720}
)

// CacheDir keeps measured modes so the probe does not have to run on every start.
const CacheDir = "cache"

// ChooseModes returns the camera modes: saved ones, freshly measured, or unmeasured presets
// when the user skips the probe or it fails.
func ChooseModes(cam Config) []Mode {
	if cached, ok := loadModes(cam); ok {
		prompt := fmt.Sprintf("\nUse camera modes measured %s? (y = use saved, n = measure again)", cached.Measured.Format("2006-01-02 15:04"))
		if askYesNo(prompt) {
			return cached.Modes
		}
	}

	if !askYesNo("\nMeasure camera modes now? Takes about a minute (y = measure, n = skip)") {
		fmt.Println("Probe skipped: using preset modes, not all of them may be supported")
		return PresetModes()
	}

	fmt.Println("Keep normal lighting: in the dark webcams lower the frame rate on their own.")

	modes, err := ProbeModes(cam.Index, cam.API)
	if err != nil || len(modes) == 0 {
		slog.Warn("could not probe camera modes, using presets", "err", err)
		fmt.Println("Probe failed: using preset modes")
		return PresetModes()
	}

	if err := saveModes(cam, modes); err != nil {
		slog.Error("save camera modes", "err", err)
	}
	return modes
}

// ProbeModes opens the camera once and measures which resolutions, formats and frame rates
// it really delivers: every frame rate is set and then timed on real frames, the value
// the driver reports is kept only for comparison.
func ProbeModes(index int, api API) ([]Mode, error) {
	cam, err := Open(Config{Index: index, API: api})
	if err != nil {
		return nil, err
	}
	defer cam.Close()

	frame := gocv.NewMat()
	defer frame.Close()

	start := time.Now()
	seen := map[string]bool{}
	var modes []Mode

	// the default format goes first: once a format is forced it cannot be reset to the default
	for _, r := range Resolutions {
		if m, ok := probeMode(cam, &frame, r, "", seen); ok {
			modes = append(modes, m)
		}
	}
	for _, f := range formatProbes {
		if m, ok := probeMode(cam, &frame, formatProbeResolution, f, seen); ok {
			modes = append(modes, m)
		}
	}

	fmt.Printf("\r%-60s\rModes probed in %s\n", "", time.Since(start).Round(time.Second))

	sortModes(modes)
	return modes, nil
}

// probeMode measures one resolution in one pixel format. Returns false when the driver
// substituted a resolution that was already measured or delivered no frames.
func probeMode(cam *Camera, frame *gocv.Mat, r Resolution, format string, seen map[string]bool) (Mode, bool) {
	if format != "" {
		cam.Set(gocv.VideoCaptureFOURCC, float64(FourCC(format)))
	}
	cam.Set(gocv.VideoCaptureFrameWidth, float64(r.Width))
	cam.Set(gocv.VideoCaptureFrameHeight, float64(r.Height))

	w, h, ok := readFrameSize(cam, frame)
	if !ok {
		return Mode{}, false
	}

	key := fmt.Sprintf("%dx%d/%s", w, h, format)
	if seen[key] {
		return Mode{}, false
	}
	seen[key] = true

	m := Mode{Width: w, Height: h, Format: format, ActualFormat: cam.Format()}

	for _, fps := range FPSPresets {
		fmt.Printf("\r%-60s", fmt.Sprintf("Probing %s @ %g FPS", m.String(), fps))

		cam.Set(gocv.VideoCaptureFPS, fps)
		res := FPSResult{
			Requested: fps,
			Declared:  math.Round(cam.ActualFPS()*100) / 100,
		}

		measured, mw, mh := measureFPS(cam, frame)
		res.Measured = math.Round(measured*100) / 100
		res.Resized = measured > 0 && (mw != w || mh != h)

		m.Results = append(m.Results, res)
	}

	m.FPS = confirmedFPS(m.Results)
	return m, true
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

// measureFPS drops a few frames after a settings change, then times up to measureFrames
// frames within measureTimeout. Returns the frame rate and the size of the last frame.
func measureFPS(cam *Camera, frame *gocv.Mat) (float64, int, int) {
	for i := 0; i < skipFrames; i++ {
		cam.Read(frame)
	}

	var first, last time.Time
	n := 0
	deadline := time.Now().Add(measureTimeout)

	for n < measureFrames && time.Now().Before(deadline) {
		if !cam.Read(frame) || frame.Empty() {
			continue
		}
		last = time.Now()
		if n == 0 {
			first = last
		}
		n++
	}

	if n < 2 {
		return 0, frame.Cols(), frame.Rows()
	}
	return float64(n-1) / last.Sub(first).Seconds(), frame.Cols(), frame.Rows()
}

// sortModes orders modes by frame size, the default format before forced ones.
func sortModes(modes []Mode) {
	sort.SliceStable(modes, func(i, j int) bool {
		a, b := modes[i], modes[j]
		if a.Width*a.Height != b.Width*b.Height {
			return a.Width*a.Height < b.Width*b.Height
		}
		if a.Width != b.Width {
			return a.Width < b.Width
		}
		return a.Format < b.Format
	})
}

// modeCache is the saved result of a probe.
type modeCache struct {
	Camera      string    `json:"camera"`
	Fingerprint string    `json:"fingerprint"`
	Measured    time.Time `json:"measured"`
	Modes       []Mode    `json:"modes"`
}

// fingerprint identifies a camera by its default mode, because camera names are only
// positions ("Camera 1") and a different device may take the same position later.
func fingerprint(cam Config) string {
	return fmt.Sprintf("%dx%d@%g/%s", cam.Width, cam.Height, cam.FPS, cam.Format)
}

var unsafeName = regexp.MustCompile(`[^A-Za-z0-9_-]+`)

// cachePath returns the cache file for a camera; a backend chosen explicitly gets its own file,
// because backends offer different modes for the same device.
func cachePath(cam Config) string {
	name := cam.Name
	if cam.API != APIAny {
		name += "_" + BackendName(cam.API)
	}
	return filepath.Join(CacheDir, "modes_"+strings.Trim(unsafeName.ReplaceAllString(name, "_"), "_")+".json")
}

// loadModes reads saved modes for the camera; false when there are none or they belong to another device.
func loadModes(cam Config) (modeCache, bool) {
	data, err := os.ReadFile(cachePath(cam))
	if err != nil {
		return modeCache{}, false
	}

	var c modeCache
	if err := json.Unmarshal(data, &c); err != nil {
		slog.Warn("camera modes cache unreadable", "err", err)
		return modeCache{}, false
	}
	if c.Fingerprint != fingerprint(cam) || len(c.Modes) == 0 {
		return modeCache{}, false
	}
	return c, true
}

// saveModes stores measured modes for the camera.
func saveModes(cam Config, modes []Mode) error {
	if err := os.MkdirAll(CacheDir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(modeCache{
		Camera:      cam.Name,
		Fingerprint: fingerprint(cam),
		Measured:    time.Now(),
		Modes:       modes,
	}, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(cachePath(cam), data, 0644)
}
