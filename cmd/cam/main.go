package main

import (
	"fmt"
	"log"
	"runtime"
	"webcam/internal/camera"
	"webcam/internal/pipeline"

	"gocv.io/x/gocv"
)

func main() {
	runtime.LockOSThread()

	cfg := camera.Config{
		Index:  1,
		API:    camera.APIAny,
		Width:  1280,
		Height: 720,
		FPS:    30,
	}

	cam, err := camera.Open(cfg)
	if err != nil {
		log.Fatalf("camera not opened: %v", err)
	}
	defer cam.Close()

	win := gocv.NewWindow("Pipeline Demo")
	defer win.Close()

	done := make(chan struct{})

	pipe := pipeline.New(
		done,
		pipeline.GrayStage(),
		pipeline.BrightnessContrastStage(1.0, 20.0),
		pipeline.ToBGRStage(),
	)

	frames := pipe.Run(pipeline.ReadStage(cam, done))
	sink := pipeline.WindowSink(win)

	fmt.Println("Press ESC to exit")

	for {
		select {
		case f, ok := <-frames:
			if !ok {
				close(done)
				return
			}
			sink(f)

		default:
			if win.WaitKey(1) == 27 {
				close(done)
				return
			}
		}
	}
}
