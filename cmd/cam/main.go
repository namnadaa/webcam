package main

import (
	"image"
	"image/color"
	"log"
	"webcam/internal/camera"

	"gocv.io/x/gocv"
)

func main() {
	cfg := camera.Config{
		Index:  0,
		API:    camera.APIAVFoundation,
		Width:  1280,
		Height: 720,
		FPS:    60,
	}

	cam, err := camera.Open(cfg)
	if err != nil {
		log.Fatalf("camera not opened on API: %v, err: %v", cfg.API, err)
	}
	defer cam.Close()

	win := gocv.NewWindow("Mac Camera")
	defer win.Close()

	frame := gocv.NewMat()
	defer frame.Close()

	for {
		ok := cam.Read(&frame)
		if !ok || frame.Empty() {
			continue
		}

		err = gocv.Rectangle(&frame, image.Rect(10, 10, 300, 60), color.RGBA{0, 0, 0, 120}, -1)
		if err != nil {
			log.Fatalf("failed to create rectangle: %v", err)
		}
		gocv.PutText(&frame, "Press ESC to exit",
			image.Pt(20, 45), gocv.FontHersheySimplex, 0.8,
			color.RGBA{0, 255, 0, 255}, 2)

		err = win.IMShow(frame)
		if err != nil {
			log.Fatalf("failed to show display: %v", err)
		}
		if win.WaitKey(1) == 27 {
			break
		}
	}
}
