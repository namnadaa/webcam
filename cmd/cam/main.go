package main

import (
	"fmt"
	"image"
	"image/color"
	"log"
	"runtime"
	"sync"
	"time"
	"webcam/internal/camera"
	"webcam/internal/control"
	"webcam/internal/control/keyboard"
	"webcam/internal/pipeline"

	"gocv.io/x/gocv"
)

func main() {
	runtime.LockOSThread()

	for {
		cams := camera.FindCameras()
		if len(cams) == 0 {
			fmt.Println("No cameras found")
			return
		}

		camera.CompareCameras(cams)
		camera.PrintCameraList(cams)

		selected, ok := camera.SelectCamera(cams)
		if !ok {
			fmt.Println("Exiting program")
			return
		}

		width, height, fps, changed := camera.AskCameraSettings()

		cfg := camera.Config{
			Index:  selected.Index,
			API:    camera.APIAny,
			Width:  selected.Width,
			Height: selected.Height,
			FPS:    selected.FPS,
		}

		if changed {
			cfg.Width = width
			cfg.Height = height
			cfg.FPS = fps
		}

		cam, err := camera.Open(cfg)
		if err != nil {
			log.Fatalf("camera not opened: %v", err)
		}

		win := gocv.NewWindow("Camera")

		actualW, actualH := cam.ActualSize()
		actualFPS := cam.ActualFPS()

		fmt.Println("Applied settings:")
		fmt.Printf("Resolution: %dx%d\n", actualW, actualH)
		fmt.Printf("FPS: %.2f\n", actualFPS)

		stages := []pipeline.StageToggle{
			{
				Enabled: true,
				Build: func(p *control.PipelineParams) pipeline.StageFunc {
					return pipeline.BrightnessContrastStage(&p.BrightnessContrast)
				},
			},
			{
				Enabled: true,
				Build: func(p *control.PipelineParams) pipeline.StageFunc {
					return pipeline.BlurStage(&p.Blur)
				},
			},
			{
				Enabled: true,
				Build: func(p *control.PipelineParams) pipeline.StageFunc {
					return pipeline.EdgeStage(&p.Edge)
				},
			},
			{
				Enabled: true,
				Build: func(p *control.PipelineParams) pipeline.StageFunc {
					return pipeline.GrayStage()
				},
			},
			{
				Enabled: true,
				Build: func(p *control.PipelineParams) pipeline.StageFunc {
					return pipeline.SharpenStage()
				},
			},
		}

		// Create pipeline parameters
		params := &control.PipelineParams{
			BrightnessContrast: control.BrightnessContrastParams{
				Alpha: 1.0,
				Beta:  0,
			},
			Blur: control.BlurParams{
				Ksize: 5,
			},
			Edge: control.EdgeParams{
				Threshold1: 50,
				Threshold2: 150,
			},
		}

		var wg sync.WaitGroup

		var (
			frameCount int
			fpsCam     float64
			lastFPS    = time.Now()
		)

		runCamera := true
		for runCamera {
			done := make(chan struct{})

			frames := pipeline.BuildPipeline(cam, stages, params, done, &wg)

			for f := range frames {
				frameStart := time.Now()

				frameCount++
				elapsed := time.Since(lastFPS)

				if elapsed >= time.Second {
					fpsCam = float64(frameCount) / elapsed.Seconds()

					frameCount = 0
					lastFPS = time.Now()
				}

				func() {
					defer f.Close()

					help := []string{
						"1: Toggle Brightness/Contrast",
						"2: Toggle Blur",
						"3: Toggle Edge",
						"4: Toggle Gray",
						"5: Toggle Sharpen",
						"ESC: Camera menu",
					}

					y := 20
					for _, line := range help {
						drawText(&f.Img, line, 10, y, color.RGBA{255, 255, 255, 0})
						y += 20
					}

					settings := []string{
						fmt.Sprintf("Contrast: %.2f", params.BrightnessContrast.Alpha),
						fmt.Sprintf("Brightness: %.0f", params.BrightnessContrast.Beta),
						fmt.Sprintf("Blur Kernel: %d", params.Blur.Ksize),
						fmt.Sprintf("Edge T1: %.0f", params.Edge.Threshold1),
						fmt.Sprintf("Edge T2: %.0f", params.Edge.Threshold2),
					}

					y += 20
					for _, line := range settings {
						drawText(&f.Img, line, 10, y, color.RGBA{0, 255, 255, 0})
						y += 20
					}

					statuses := []string{
						fmt.Sprintf("[%s] Brightness/Contrast", onOff(stages[0].Enabled)),
						fmt.Sprintf("[%s] Blur", onOff(stages[1].Enabled)),
						fmt.Sprintf("[%s] Edge", onOff(stages[2].Enabled)),
						fmt.Sprintf("[%s] Gray", onOff(stages[3].Enabled)),
						fmt.Sprintf("[%s] Sharpen", onOff(stages[4].Enabled)),
					}

					y += 20
					for _, line := range statuses {
						drawText(&f.Img, line, 10, y, color.RGBA{0, 255, 0, 0})
						y += 20
					}

					latency := time.Since(frameStart).Milliseconds()

					stats := []string{
						fmt.Sprintf("FPS: %.2f", fpsCam),
						fmt.Sprintf("Latency: %d ms", latency),
						fmt.Sprintf("Resolution: %dx%d", actualW, actualH),
						fmt.Sprintf("Camera: %s", selected.Name),
					}

					statsY := 20
					for _, line := range stats {
						size := gocv.GetTextSize(line, gocv.FontHersheyPlain, 1.2, 1)
						x := f.Img.Cols() - size.X - 10
						drawText(&f.Img, line, x, statsY, color.RGBA{255, 165, 0, 0})
						statsY += 20
					}

					win.IMShow(f.Img)

					cont, changed := keyboard.HandleKeyboard(win, params, stages)
					if !cont {
						safeClose(done)
						wg.Wait()
						runCamera = false
						return
					}

					if changed {
						safeClose(done)
						wg.Wait()
						return
					}
				}()

				if !runCamera {
					break
				}
			}
		}

		_ = cam.Close()
		_ = win.Close()

		fmt.Println("\nReturn to camera selection")
	}
}

// safeClose safely closes a channel if it has not been closed yet.
func safeClose(ch chan struct{}) {
	select {
	case <-ch:
		return
	default:
		close(ch)
	}
}

// drawText renders outlined text on an image for better visibility.
func drawText(img *gocv.Mat, text string, x, y int, clr color.RGBA) {
	gocv.PutText(
		img,
		text,
		image.Pt(x, y),
		gocv.FontHersheyPlain,
		1.2,
		color.RGBA{0, 0, 0, 0},
		3,
	)

	gocv.PutText(
		img,
		text,
		image.Pt(x, y),
		gocv.FontHersheyPlain,
		1.2,
		clr,
		1,
	)
}

// onOff converts a boolean value into an ON/OFF status string.
func onOff(v bool) string {
	if v {
		return "ON "
	}
	return "OFF"
}
