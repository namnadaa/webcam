package main

import (
	"fmt"
	"log/slog"
	"runtime"
	"sync"
	"time"
	"webcam/internal/actions"
	"webcam/internal/camera"
	"webcam/internal/control"
	"webcam/internal/control/keyboard"
	"webcam/internal/helpers"
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
				break
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

			stages := pipeline.NewStages()
			params := control.NewPipelineParams()
			uiState := ui.NewUIState()
			mediaState := media.NewMediaState()

			var wg sync.WaitGroup

			var (
				frameCount int
				fpsCam     float64
				lastFPS    = time.Now()
			)

			recorder := &media.VideoRecorder{}

			runCamera := true
			for runCamera {
				done := make(chan struct{})

				frames := pipeline.BuildPipeline(cam, stages, params, done, &wg)

				for f := range frames {
					helpers.UpdateFPS(&frameCount, &lastFPS, &fpsCam)

					func() {
						defer f.Close()

						cont, pipelineChanged, action := keyboard.HandleKeyboard(win, params, stages, &uiState, &mediaState)

						if !cont {
							safeClose(done)
							wg.Wait()
							runCamera = false
							return
						}

						cameraChanged := actions.HandleAction(action, &cfg, &resolutionIndex, &fpsIndex)
						if cameraChanged {
							safeClose(done)
							wg.Wait()

							restartCamera = true
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
