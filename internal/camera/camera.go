package camera

import (
	"fmt"
	"strconv"
	"strings"

	"gocv.io/x/gocv"
)

// API defines a video capture backend type.
type API = gocv.VideoCaptureAPI

// Supported video capture backends.
const (
	APIAny          = gocv.VideoCaptureAny          // Auto-detect backend
	APIAVFoundation = gocv.VideoCaptureAVFoundation // macOS AVFoundation
	APIV4L2         = gocv.VideoCaptureV4L2         // Linux V4L2
	APIMSMF         = gocv.VideoCaptureMSMF         // Windows Media Foundation
)

// Config holds camera initialization parameters.
type Config struct {
	Index  int
	API    API
	Width  int
	Height int
	FPS    float64
	Format string // FourCC pixel format, e.g. "MJPG"; "" lets the driver choose
	Name   string
}

// Formats lists the pixel formats offered for selection.
// MJPG is compressed (more frames per second, adds compression artifacts),
// YUY2 and NV12 are uncompressed.
var Formats = []string{"MJPG", "YUY2", "NV12"}

// Camera represents an active video capture device.
type Camera struct {
	cap *gocv.VideoCapture
	cfg Config
}

// Open initializes and returns a new camera instance.
func Open(cfg Config) (*Camera, error) {
	if cfg.API == 0 {
		cfg.API = APIAny
	}

	var cap *gocv.VideoCapture
	var err error

	cap, err = gocv.OpenVideoCaptureWithAPI(cfg.Index, cfg.API)
	if err != nil {
		return nil, fmt.Errorf("failed to open capture: %w", err)
	}

	if cap == nil || !cap.IsOpened() {
		cap.Close()
		return nil, fmt.Errorf("no available camera found (API=%v)", cfg.API)
	}

	// the format must be set before the size: drivers pick the size from the formats they offer
	if cfg.Format != "" {
		cap.Set(gocv.VideoCaptureFOURCC, float64(FourCC(cfg.Format)))
	}

	if cfg.Width > 0 {
		cap.Set(gocv.VideoCaptureFrameWidth, float64(cfg.Width))
	}

	if cfg.Height > 0 {
		cap.Set(gocv.VideoCaptureFrameHeight, float64(cfg.Height))
	}

	if cfg.FPS > 0 {
		cap.Set(gocv.VideoCaptureFPS, float64(cfg.FPS))
	}

	cap.Set(gocv.VideoCaptureConvertRGB, 1)
	cap.Set(gocv.VideoCaptureBufferSize, 1)

	c := Camera{
		cap: cap,
		cfg: cfg,
	}
	return &c, nil
}

// Close releases the camera resource.
func (c *Camera) Close() error {
	err := c.cap.Close()
	if err != nil {
		return fmt.Errorf("failed to close capture: %w", err)
	}
	return nil
}

// Read captures a single frame from the camera.
func (c *Camera) Read(dst *gocv.Mat) bool {
	return c.cap.Read(dst)
}

// Get returns the current value of a camera property.
func (c *Camera) Get(prop gocv.VideoCaptureProperties) float64 {
	return c.cap.Get(prop)
}

// Set sets a camera property to a given value.
func (c *Camera) Set(prop gocv.VideoCaptureProperties, val float64) {
	c.cap.Set(prop, val)
}

// ActualSize returns the applied frame width and height.
func (c *Camera) ActualSize() (w, h int) {
	wf := c.cap.Get(gocv.VideoCaptureFrameWidth)
	hf := c.cap.Get(gocv.VideoCaptureFrameHeight)
	return int(wf + 0.5), int(hf + 0.5)
}

// ActualFPS returns the applied frames per second.
func (c *Camera) ActualFPS() float64 {
	return c.cap.Get(gocv.VideoCaptureFPS)
}

// PixelFormat returns the FOURCC pixel format of the camera stream.
func (c *Camera) PixelFormat() int {
	return int(c.cap.Get(gocv.VideoCaptureFOURCC))
}

// Format returns the applied pixel format as text, or "unknown" when the backend does not report it.
func (c *Camera) Format() string {
	if f := FourCCToString(c.PixelFormat()); f != "" {
		return f
	}
	return "unknown"
}

// FourCC converts a four-letter format name into its integer code; shorter names are padded with spaces.
func FourCC(name string) int {
	b := []byte(name + "    ")[:4]
	return int(b[0]) | int(b[1])<<8 | int(b[2])<<16 | int(b[3])<<24
}

// FourCCToString converts a FOURCC integer code into a human-readable string.
// Returns empty string if bytes are non-printable.
func FourCCToString(fourcc int) string {
	b := []byte{
		byte(fourcc),
		byte(fourcc >> 8),
		byte(fourcc >> 16),
		byte(fourcc >> 24),
	}
	for _, c := range b {
		if c < 32 || c > 126 {
			return ""
		}
	}
	return strings.TrimRight(string(b), " ")
}

// FindCameras searches for connected cameras and returns the Config list.
func FindCameras() []Config {
	var cameras []Config

	const maxIndex = 20
	const maxFails = 3

	fails := 0

	for i := 0; i < maxIndex; i++ {
		cfg := Config{
			Index: i,
			API:   APIAny,
		}

		cam, err := Open(cfg)
		if err != nil || cam == nil {
			fails++
			if fails >= maxFails {
				break
			}
			continue
		}

		fails = 0

		width, height := cam.ActualSize()
		fps := cam.ActualFPS()
		format := FourCCToString(cam.PixelFormat())
		if format == "" {
			format = "Unknown"
		}

		_ = cam.Close()

		cfg = Config{
			Index:  i,
			API:    cfg.API,
			Width:  width,
			Height: height,
			FPS:    fps,
			Format: format,
			Name:   fmt.Sprintf("Camera %d", len(cameras)+1),
		}

		cameras = append(cameras, cfg)
	}

	return cameras
}

// Settings are the camera parameters the user picked in the console.
type Settings struct {
	Width  int
	Height int
	FPS    float64
	Format string // "" = driver default
}

// AskCameraSettings optionally prompts the user to pick one of the supported modes (printed by PrintModes)
// and a pixel format. Returns false when the user keeps the camera defaults.
func AskCameraSettings(modes []Mode) (Settings, bool) {
	if len(modes) == 0 || !askYesNo("\nChange camera settings? (y/n)") {
		return Settings{}, false
	}

	m := modes[askNumber("Select mode", len(modes))-1]

	fps := 0.0
	if len(m.FPS) > 0 {
		fmt.Println()
		for i, v := range m.FPS {
			fmt.Printf("[%d] %g FPS\n", i+1, v)
		}
		fps = m.FPS[askNumber("Select FPS", len(m.FPS))-1]
	}

	format := m.Format
	if format == "" {
		format = askFormat()
	}

	return Settings{Width: m.Width, Height: m.Height, FPS: fps, Format: format}, true
}

// askYesNo prints the prompt and repeats until the answer is 'y' or 'n'.
func askYesNo(prompt string) bool {
	var choice string

	for {
		fmt.Print(prompt + ": ")
		fmt.Scan(&choice)

		switch strings.ToLower(strings.TrimSpace(choice)) {
		case "y":
			return true
		case "n":
			return false
		}
		fmt.Println("Please enter 'y' or 'n'")
	}
}

// askFormat lets the user pick a pixel format or keep the driver default.
func askFormat() string {
	fmt.Println("\n[1] auto (driver default)")
	for i, f := range Formats {
		fmt.Printf("[%d] %s\n", i+2, f)
	}

	n := askNumber("Select format", len(Formats)+1)
	if n == 1 {
		return ""
	}
	return Formats[n-2]
}

// askNumber reads a number in range [1, max] from stdin, repeating until the input is valid.
func askNumber(prompt string, max int) int {
	var input string

	for {
		fmt.Printf("%s [1-%d]: ", prompt, max)
		fmt.Scan(&input)

		n, err := strconv.Atoi(strings.TrimSpace(input))
		if err == nil && n >= 1 && n <= max {
			return n
		}
		fmt.Printf("Please enter a number from 1 to %d\n", max)
	}
}
