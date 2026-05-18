package main

import (
	"fmt"
	"image"
	"image/color"
	"log"
	"runtime"
	"sync"
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

		width, height, fps, ok := camera.AskCameraSettings()

		cfg := camera.Config{
			Index:  selected.Index,
			API:    camera.APIAny,
			Width:  selected.Width,
			Height: selected.Height,
			FPS:    selected.FPS,
		}

		if ok {
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
		runCamera := true
		for runCamera {
			done := make(chan struct{})

			frames := pipeline.BuildPipeline(cam, stages, params, done, &wg)

			for f := range frames {
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
						gocv.PutText(
							&f.Img,
							line,
							image.Pt(10, y),
							gocv.FontHersheyPlain,
							1.2,
							color.RGBA{255, 255, 255, 0},
							1,
						)
						y += 20
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

func safeClose(ch chan struct{}) {
	select {
	case <-ch:
		return
	default:
		close(ch)
	}
}
