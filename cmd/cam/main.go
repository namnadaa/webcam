package main

import (
	"fmt"
	"image"
	"image/color"
	"log"
	"runtime"
	"webcam/internal/camera"
	"webcam/internal/filters"

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

	win := gocv.NewWindow("Interactive Filters")
	defer win.Close()

	frame := gocv.NewMat()
	defer frame.Close()

	alpha := 1.0 // contrast
	beta := 0.0  // brightness

	fmt.Println("Controls:")
	fmt.Println("  ↑ / w : increase brightness")
	fmt.Println("  ↓ / s : decrease brightness")
	fmt.Println("  → / d : increase contrast")
	fmt.Println("  ← / a : decrease contrast")
	fmt.Println("  ESC   : exit")

	for {
		if ok := cam.Read(&frame); !ok || frame.Empty() {
			continue
		}

		gray, _ := filters.Gray(frame)
		processed, _ := filters.BrightnessContrast(gray, alpha, beta)

		err := gocv.Rectangle(&processed, image.Rect(10, 10, 360, 80), color.RGBA{0, 0, 0, 120}, -1)
		if err != nil {
			log.Fatalf("failed to create rectangle: %v", err)
		}

		text := fmt.Sprintf("contrast=%.1f  brightness=%.1f", alpha, beta)
		gocv.PutText(&processed, text, image.Pt(20, 50),
			gocv.FontHersheySimplex, 0.7, color.RGBA{255, 255, 255, 0}, 2)

		win.IMShow(processed)

		key := win.WaitKey(10)
		switch key {
		case 27:
			return
		case int('w'), int('W'):
			beta += 5
		case int('s'), int('S'):
			beta -= 5
		case int('d'), int('D'):
			alpha += 0.1
		case int('a'), int('A'):
			alpha -= 0.1
		}

		if alpha < 0.1 {
			alpha = 0.1
		}
		if alpha > 3.0 {
			alpha = 3.0
		}
		if beta < -100 {
			beta = -100
		}
		if beta > 100 {
			beta = 100
		}
	}
}
