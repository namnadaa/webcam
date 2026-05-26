package main

import (
	"fmt"
	"log/slog"
	"os"
	"runtime"
	"sync"
	"time"
	"webcam/internal/camera"
	"webcam/internal/control"
	"webcam/internal/control/keyboard"
	"webcam/internal/logger"
	"webcam/internal/media"
	"webcam/internal/pipeline"
	"webcam/internal/ui"

	"gocv.io/x/gocv"
)

func main() {
	err := logger.Init()
	if err != nil {
		slog.Error("camera not opened", "err", err)
		return
	}

	runtime.LockOSThread()

	for {
		cams := camera.FindCameras()
		if len(cams) == 0 {
			slog.Warn("No cameras found")
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

		resolutionIndex := 0
		fpsIndex := 0

		restartCamera := false

		for {

			if changed {
				cfg.Width = width
				cfg.Height = height
				cfg.FPS = fps
			}

			cam, err := camera.Open(cfg)
			if err != nil {
				slog.Error("camera not opened", "err", err)
				os.Exit(1)
			}

			win := gocv.NewWindow("Camera")

			actualW, actualH := cam.ActualSize()
			actualFPS := cam.ActualFPS()

			slog.Info(
				"camera settings applied",
				"width", actualW,
				"height", actualH,
				"fps", actualFPS,
			)

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

			cameraChanged := false
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

						cont, pipelineChanged, action := keyboard.HandleKeyboard(win, params, stages, &uiState, &mediaState)

						if !cont {
							safeClose(done)
							wg.Wait()
							runCamera = false
							return
						}

						switch action {
						case keyboard.ActionNextResolution:
							resolutionIndex++

							if resolutionIndex >= len(camera.Resolutions) {
								resolutionIndex = 0
							}

							cameraChanged = true

						case keyboard.ActionPrevResolution:
							resolutionIndex--

							if resolutionIndex < 0 {
								resolutionIndex = len(camera.Resolutions) - 1
							}

							cameraChanged = true

						case keyboard.ActionNextFPS:
							fpsIndex++

							if fpsIndex >= len(camera.FPSPresets) {
								fpsIndex = 0
							}

							cameraChanged = true

						case keyboard.ActionPrevFPS:
							fpsIndex--

							if fpsIndex < 0 {
								fpsIndex = len(camera.FPSPresets) - 1
							}

							cameraChanged = true
						}

						if cameraChanged {
							resolution := camera.Resolutions[resolutionIndex]

							cfg.Width = resolution.Width
							cfg.Height = resolution.Height
							cfg.FPS = camera.FPSPresets[fpsIndex]

							cameraChanged = false
							restartCamera = true

							safeClose(done)
							wg.Wait()

							runCamera = false
							return
						}

						event := media.UpdateRecorder(recorder, &mediaState, actualFPS, actualW, actualH)

						switch event {
						case media.RecorderStarted:
							uiState.ShowNotification("Recording started", 2*time.Second)
						case media.RecorderStopped:
							uiState.ShowNotification("Recording stopped", 2*time.Second)
						}

						if media.HandleScreenshot(&mediaState, f.Img) {
							uiState.ShowNotification("Screenshot saved", 2*time.Second)
						}

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

						if pipelineChanged {
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

			if restartCamera {
				restartCamera = false
				continue
			}

			break
		}

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
