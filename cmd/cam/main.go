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
	"webcam/internal/graph"
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

		settings, changed := camera.AskCameraSettings(modes)
		steps := matrixSteps(camera.AskMatrix(modes))

		cfg := camera.Config{
			Index:  selected.Index,
			API:    camera.APIAny,
			Width:  selected.Width,
			Height: selected.Height,
			FPS:    selected.FPS,
		}

		if changed {
			cfg.Width = settings.Width
			cfg.Height = settings.Height
			cfg.FPS = settings.FPS
			cfg.Format = settings.Format
		}

		resolutionIndex, fpsIndex := camera.FindMode(modes, cfg.Width, cfg.Height, cfg.FPS)

		restartCamera := false

		autoLocked := false
		var savedControls camera.Controls

		exp := experiment.NewController()
		var noiseGraph *graph.Graph

		var matrix *experiment.Matrix
		var matrixHome camera.Config // camera settings to return to after the matrix
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
			format := cam.Format()
			if cfg.Format != "" && format != cfg.Format {
				fmt.Printf("\nWarning: requested format %s, camera uses %s\n", cfg.Format, format)
				slog.Warn("pixel format not applied", "requested", cfg.Format, "actual", format)
			}

			if autoLocked {
				cam.LockAuto()
			}

			exp.ResetMeter()
			openedAt := time.Now()

			slog.Info(
				"camera settings applied",
				"width", actualW,
				"height", actualH,
				"fps", actualFPS,
				"format", format,
				"requested_format", cfg.Format,
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

						if matrix != nil && blockedDuringMatrix(action) {
							uiState.ShowNotification("Mode matrix running, [U] aborts it", 2*time.Second)
							action = actions.ActionNone
						}

						switchCamera := false

						switch action {
						case actions.ActionToggleMatrix:
							switch {
							case matrix != nil:
								if exp.Running() {
									exp.Stop()
								}
								saveMatrix(matrix, true)
								matrix = nil
								cfg = matrixHome
								resolutionIndex, fpsIndex = camera.FindMode(modes, cfg.Width, cfg.Height, cfg.FPS)
								switchCamera = true
							case len(steps) == 0:
								uiState.ShowNotification("No modes selected for the matrix at startup", 3*time.Second)
							case exp.Running():
								uiState.ShowNotification("Stop the experiment first ([B])", 3*time.Second)
							default:
								matrixHome = cfg
								matrix = experiment.NewMatrix(steps, exp.Config)
								applyStep(&cfg, matrix.Current())
								fmt.Printf("\nMode matrix started: %d modes, %s\n", len(steps), exp.Config.String())
								switchCamera = true
							}
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
							meta := experimentMeta(cam, f, selected.Name, actualFPS, format, cfg.Format, autoLocked)
							uiState.ShowNotification(exp.Toggle(meta), 5*time.Second)
						case actions.ActionToggleGraph:
							uiState.ShowNotification(toggleGraph(&noiseGraph), 3*time.Second)
						}

						cameraChanged := switchCamera || actions.HandleAction(action, modes, &cfg, &resolutionIndex, &fpsIndex)
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

						msg, finished := exp.Update(f.Raw, f.Time)
						if msg != "" {
							uiState.ShowNotification(msg, 8*time.Second)
						}

						if matrix != nil && finished != nil {
							if matrix.Record(*finished) {
								applyStep(&cfg, matrix.Current())
							} else {
								saveMatrix(matrix, false)
								matrix = nil
								cfg = matrixHome
								resolutionIndex, fpsIndex = camera.FindMode(modes, cfg.Width, cfg.Height, cfg.FPS)
							}

							safeClose(done)
							wg.Wait()

							restartCamera = true
							runCamera = false
							return
						}

						if matrix != nil && !exp.Running() && time.Since(openedAt) >= experiment.Warmup {
							meta := experimentMeta(cam, f, selected.Name, actualFPS, format, cfg.Format, autoLocked)
							fmt.Printf("\nMode matrix %s: %s\n", matrix.Progress(), matrix.Current())
							uiState.ShowNotification(exp.Toggle(meta), 3*time.Second)
						}

						if noiseGraph != nil {
							last := exp.Last()
							noiseGraph.Add(graph.Point{
								Time:     last.Time,
								Mean:     last.Mean,
								Noise:    last.Noise,
								HasNoise: last.HasNoise,
							}, exp.Config.Threshold)
						}

						ui.RenderOverlay(
							&f.Img,
							uiState,
							params,
							stages,
							fpsCam,
							actualW,
							actualH,
							format,
							selected.Name,
							f.Time,
							experimentInfo(exp, matrix),
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
		if matrix != nil {
			saveMatrix(matrix, true)
		}
		if noiseGraph != nil {
			toggleGraph(&noiseGraph)
		}

		fmt.Println("\nReturn to camera selection")
	}
}

// experimentMeta describes the current capture conditions for an experiment report.
func experimentMeta(cam *camera.Camera, f pipeline.Frame, cameraName string, fps float64, format, requested string, autoLocked bool) experiment.Meta {
	return experiment.Meta{
		Camera:     cameraName,
		Width:      f.Raw.Cols(),
		Height:     f.Raw.Rows(),
		FPS:        fps,
		Format:     format,
		Requested:  requested,
		AutoLocked: autoLocked,
		Controls:   cam.ReadControls().String(),
	}
}

// matrixSteps expands the selected resolutions into camera modes, one per frame rate.
func matrixSteps(modes []camera.Mode) []experiment.Step {
	var steps []experiment.Step
	for _, m := range modes {
		if len(m.FPS) == 0 {
			steps = append(steps, experiment.Step{Width: m.Width, Height: m.Height})
			continue
		}
		for _, fps := range m.FPS {
			steps = append(steps, experiment.Step{Width: m.Width, Height: m.Height, FPS: fps})
		}
	}
	return steps
}

// applyStep switches the camera config to a matrix step.
func applyStep(cfg *camera.Config, step experiment.Step) {
	cfg.Width = step.Width
	cfg.Height = step.Height
	cfg.FPS = step.FPS
}

// saveMatrix writes the matrix summary, logging a failure.
func saveMatrix(m *experiment.Matrix, aborted bool) {
	if _, err := m.Save(aborted); err != nil {
		slog.Error("mode matrix save", "err", err)
	}
}

// blockedDuringMatrix reports whether the action would interfere with a running mode matrix.
func blockedDuringMatrix(action actions.CameraAction) bool {
	switch action {
	case actions.ActionNextResolution, actions.ActionPrevResolution,
		actions.ActionNextFPS, actions.ActionPrevFPS,
		actions.ActionToggleExperiment, actions.ActionNextExperimentMode,
		actions.ActionNextExperimentDuration, actions.ActionThresholdUp, actions.ActionThresholdDown:
		return true
	}
	return false
}

// experimentInfo collects what the on-screen experiment panel shows.
func experimentInfo(exp *experiment.Controller, matrix *experiment.Matrix) ui.ExperimentInfo {
	last := exp.Last()

	matrixText := ""
	if matrix != nil {
		matrixText = matrix.Progress() + " " + matrix.Current().String()
		if !exp.Running() {
			matrixText += " warming up"
		}
	}

	return ui.ExperimentInfo{
		Mode:      exp.Config.Mode.String(),
		Duration:  experiment.FormatDuration(exp.Config.Duration()),
		Threshold: exp.Config.Threshold,
		Running:   exp.Running(),
		Elapsed:   experiment.FormatClock(exp.Elapsed()),
		Noise:     last.Noise,
		HasNoise:  last.HasNoise,
		Level:     last.Mean,
		Light:     exp.LightState(),
		Matrix:    matrixText,
	}
}

// toggleGraph opens the real-time noise graph window or closes it.
// Returns a message for the on-screen notification.
func toggleGraph(g **graph.Graph) string {
	if *g != nil {
		path := (*g).Path()
		if err := (*g).Close(); err != nil {
			slog.Error("noise graph close", "err", err)
		}
		*g = nil
		return "Noise graph closed, data: " + path
	}

	opened, err := graph.Open()
	if err != nil {
		slog.Error("noise graph open", "err", err)
		return "Noise graph error: " + err.Error()
	}
	*g = opened
	fmt.Printf("\nNoise graph logging to %s\n", opened.Path())
	return "Noise graph opened, logging to " + opened.Path()
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
