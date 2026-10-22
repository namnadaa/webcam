package main

import (
	"fmt"
	"log/slog"
	"runtime"
	"strings"
	"sync"
	"time"
	"webcam/internal/actions"
	"webcam/internal/camera"
	"webcam/internal/control"
	"webcam/internal/control/keyboard"
	"webcam/internal/experiment"
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

		fmt.Println("\nProbing supported modes...")
		modes, err := camera.ProbeModes(selected.Index, camera.APIAny)
		if err != nil || len(modes) == 0 {
			slog.Warn("could not probe camera modes, using presets", "err", err)
			modes = camera.PresetModes()
		}
		camera.PrintModes(modes)

		width, height, fps, changed := camera.AskCameraSettings(modes)

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

		resolutionIndex, fpsIndex := camera.FindMode(modes, cfg.Width, cfg.Height, cfg.FPS)

		restartCamera := false

		autoLocked := false
		var savedControls camera.Controls

		exp := experiment.NewController()
		fmt.Printf("\nNoise threshold: %.1f DN (change with -/= in the camera window)\n", exp.Config.Threshold)

		for {
			cam, err := camera.Open(cfg)
			if err != nil {
				slog.Error("camera not opened", "err", err)
				break
			}

			win := gocv.NewWindow("Camera")

			actualW, actualH := cam.ActualSize()
			actualFPS := cam.ActualFPS()
			format := camera.FourCCToString(cam.PixelFormat())

			if autoLocked {
				cam.LockAuto()
			}

			exp.ResetMeter()

			slog.Info(
				"camera settings applied",
				"width", actualW,
				"height", actualH,
				"fps", actualFPS,
				"format", format,
				"controls", cam.ReadControls().String(),
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

						switch action {
						case actions.ActionToggleAutoLock:
							uiState.ShowNotification(toggleAutoLock(cam, &autoLocked, &savedControls), 3*time.Second)
						case actions.ActionNextExperimentMode:
							uiState.ShowNotification(exp.NextMode(), 3*time.Second)
						case actions.ActionNextExperimentDuration:
							uiState.ShowNotification(exp.NextDuration(), 2*time.Second)
						case actions.ActionThresholdUp:
							uiState.ShowNotification(exp.AdjustThreshold(experiment.ThresholdStep), 2*time.Second)
						case actions.ActionThresholdDown:
							uiState.ShowNotification(exp.AdjustThreshold(-experiment.ThresholdStep), 2*time.Second)
						case actions.ActionToggleExperiment:
							meta := experiment.Meta{
								Camera:     selected.Name,
								Width:      f.Raw.Cols(),
								Height:     f.Raw.Rows(),
								FPS:        actualFPS,
								Format:     format,
								AutoLocked: autoLocked,
								Controls:   cam.ReadControls().String(),
							}
							uiState.ShowNotification(exp.Toggle(meta), 5*time.Second)
						}

						cameraChanged := actions.HandleAction(action, modes, &cfg, &resolutionIndex, &fpsIndex)
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

						if msg := exp.Update(f.Raw, f.Time); msg != "" {
							uiState.ShowNotification(msg, 8*time.Second)
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
							experimentInfo(exp),
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

			if exp.Running() {
				exp.Stop()
			}

			_ = cam.Close()
			_ = win.Close()

			if restartCamera {
				restartCamera = false
				continue
			}

			break
		}

		exp.Close()

		fmt.Println("\nReturn to camera selection")
	}
}

// experimentInfo collects what the on-screen experiment panel shows.
func experimentInfo(exp *experiment.Controller) ui.ExperimentInfo {
	last := exp.Last()
	return ui.ExperimentInfo{
		Mode:      exp.Config.Mode.String(),
		Duration:  experiment.FormatDuration(exp.Config.Duration()),
		Threshold: exp.Config.Threshold,
		Running:   exp.Running(),
		Elapsed:   experiment.FormatClock(exp.Elapsed()),
		Noise:     last.Noise,
		HasNoise:  last.HasNoise,
		Level:     last.Mean,
	}
}

// toggleAutoLock locks or restores the camera's automatic exposure, white balance and focus.
// Returns a message for the on-screen notification.
func toggleAutoLock(cam *camera.Camera, locked *bool, saved *camera.Controls) string {
	if *locked {
		cam.UnlockAuto(*saved)
		*locked = false
		slog.Info("auto controls restored", "controls", cam.ReadControls().String())
		return "Auto controls restored"
	}

	before, after := cam.LockAuto()
	slog.Info("auto controls lock", "before", before.String(), "after", after.String())

	applied := camera.LockApplied(before, after)
	if len(applied) == 0 {
		return "Auto controls lock not supported by camera backend"
	}

	*saved = before
	*locked = true
	return "Locked: " + strings.Join(applied, ", ")
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
