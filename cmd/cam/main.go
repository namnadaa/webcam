package main

import (
	"fmt"
	"log"
	"runtime"
	"sync"
	"time"
	"webcam/internal/camera"
	"webcam/internal/control"
	"webcam/internal/control/keyboard"
	"webcam/internal/media"
	"webcam/internal/pipeline"
	"webcam/internal/ui"

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
				Enabled: false,
				Build: func(p *control.PipelineParams) pipeline.StageFunc {
					return pipeline.BrightnessContrastStage(&p.BrightnessContrast)
				},
			},
			{
				Enabled: false,
				Build: func(p *control.PipelineParams) pipeline.StageFunc {
					return pipeline.BlurStage(&p.Blur)
				},
			},
			{
				Enabled: false,
				Build: func(p *control.PipelineParams) pipeline.StageFunc {
					return pipeline.EdgeStage(&p.Edge)
				},
			},
			{
				Enabled: false,
				Build: func(p *control.PipelineParams) pipeline.StageFunc {
					return pipeline.GrayStage()
				},
			},
			{
				Enabled: false,
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

		uiState := ui.State{
			ShowMenu:     true,
			ShowStats:    false,
			ShowSettings: false,
			ShowStatuses: false,
			ShowControls: false,
		}

		mediaState := media.State{
			Screenshot: false,
		}
		recorder := &media.VideoRecorder{}

		runCamera := true
		for runCamera {
			done := make(chan struct{})

			frames := pipeline.BuildPipeline(cam, stages, params, done, &wg)

			for f := range frames {
				frameCount++
				elapsed := time.Since(lastFPS)

				if elapsed >= time.Second {
					fpsCam = float64(frameCount) / elapsed.Seconds()

					frameCount = 0
					lastFPS = time.Now()
				}

				func() {
					defer f.Close()

					cont, changed := keyboard.HandleKeyboard(win, params, stages, &uiState, &mediaState)

					if !cont {
						safeClose(done)
						wg.Wait()
						runCamera = false
						return
					}

					media.UpdateRecorder(recorder, &mediaState, actualFPS, actualW, actualH)

					media.HandleScreenshot(&mediaState, f.Img)

					ui.RenderOverlay(
						&f.Img,
						uiState,
						params,
						stages,
						fpsCam,
						actualW,
						actualH,
						selected.Name,
						f.Time,
					)

					recorder.Write(f.Img)

					win.IMShow(f.Img)

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
