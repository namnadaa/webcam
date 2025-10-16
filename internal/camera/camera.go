package camera

import (
	"fmt"

	"gocv.io/x/gocv"
)

type API = gocv.VideoCaptureAPI

const (
	APIAny          = gocv.VideoCaptureAny
	APIAVFoundation = gocv.VideoCaptureAVFoundation
	APIV4L2         = gocv.VideoCaptureV4L2
	APIMSMF         = gocv.VideoCaptureMSMF
)

type Config struct {
	Index  int
	API    API
	Width  int
	Height int
	FPS    float64
}

type Camera struct {
	cap *gocv.VideoCapture
	cfg Config
}

func Open(cfg Config) (*Camera, error) {
	if cfg.API == 0 {
		cfg.API = APIAny
	}

	var cap *gocv.VideoCapture
	var err error

	for i := 0; i < 5; i++ {
		cap, err = gocv.OpenVideoCaptureWithAPI(i, cfg.API)
		if err != nil {
			return nil, fmt.Errorf("failed to open capture: %w", err)
		}

		if cap.IsOpened() {
			fmt.Printf("Camera opened on index %v\n", i)
			cfg.Index = i
			break
		}
	}

	if cap == nil || !cap.IsOpened() {
		cap.Close()
		return nil, fmt.Errorf("no available camera found (API=%v)", cfg.API)
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

func (c *Camera) Close() error {
	err := c.cap.Close()
	if err != nil {
		return fmt.Errorf("failed to close capture: %w", err)
	}
	return nil
}

func (c *Camera) Read(dst *gocv.Mat) bool {
	return c.cap.Read(dst)
}
