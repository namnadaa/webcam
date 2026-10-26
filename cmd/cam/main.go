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
	"webcam/internal/i18n"
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

	i18n.Ask()
	api := camera.ChooseBackend()

	for {
		cams := camera.FindCameras(api)
		if len(cams) == 0 {
			slog.Warn("No cameras found")
			fmt.Println(i18n.T("No cameras found"))
			return
		}

		camera.CompareCameras(cams)
		camera.PrintCameraList(cams)

		selected, ok := camera.SelectCamera(cams)
		if !ok {
			fmt.Println(i18n.T("Exiting program"))
			return
		}

		modes := camera.ChooseModes(*selected)
		camera.PrintModes(modes)

		settings, changed := camera.AskCameraSettings(modes)
		steps := matrixSteps(camera.AskMatrix(modes))

		cfg := camera.Config{
			Index:  selected.Index,
			API:    selected.API,
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

		resolutionIndex, fpsIndex := camera.FindMode(modes, cfg.Width, cfg.Height, cfg.FPS, cfg.Format)

		restartCamera := false

		autoLocked := false
		var savedControls camera.Controls

		exp := experiment.NewController()
		var noiseGraph *graph.Graph

		var matrix *experiment.Matrix
		var matrixHome camera.Config // camera settings to return to after the matrix
		fmt.Print(i18n.Tf("\nNoise threshold: %.1f DN (change with -/= in the camera window)\n", exp.Config.Threshold))

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
				fmt.Print(i18n.Tf("\nWarning: requested format %s, camera uses %s\n", cfg.Format, format))
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
				"backend", cam.Backend(),
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
							uiState.ShowNotification(i18n.T("Mode matrix running, [U] aborts it"), 2*time.Second)
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
								resolutionIndex, fpsIndex = camera.FindMode(modes, cfg.Width, cfg.Height, cfg.FPS, cfg.Format)
								switchCamera = true
							case len(steps) == 0:
								uiState.ShowNotification(i18n.T("No modes selected for the matrix at startup"), 3*time.Second)
							case exp.Running():
								uiState.ShowNotification(i18n.T("Stop the experiment first ([B])"), 3*time.Second)
							default:
								matrixHome = cfg
								matrix = experiment.NewMatrix(steps, exp.Config)
								applyStep(&cfg, matrix.Current())
								fmt.Print(i18n.Tf("\nMode matrix started: %d modes, %s\n", len(steps), exp.Config.String()))
								switchCamera = true
							}
						case actions.ActionToggleAutoLock:
							uiState.ShowNotification(toggleAutoLock(cam, &autoLocked, &savedControls), 5*time.Second)
						case actions.ActionDriverSettings:
							uiState.ShowNotification(openDriverSettings(cam), 5*time.Second)
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
							uiState.ShowNotification(i18n.T("Recording started"), 2*time.Second)
						case media.RecorderStopped:
							uiState.ShowNotification(i18n.T("Recording stopped"), 2*time.Second)
						}

						if media.HandleScreenshot(&mediaState, f.Img) {
							uiState.ShowNotification(i18n.T("Screenshot saved"), 2*time.Second)
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
								resolutionIndex, fpsIndex = camera.FindMode(modes, cfg.Width, cfg.Height, cfg.FPS, cfg.Format)
							}

							safeClose(done)
							wg.Wait()

							restartCamera = true
							runCamera = false
							return
						}

						if matrix != nil && !exp.Running() && time.Since(openedAt) >= experiment.Warmup {
							meta := experimentMeta(cam, f, selected.Name, actualFPS, format, cfg.Format, autoLocked)
							fmt.Print(i18n.Tf("\nMode matrix %s: %s\n", matrix.Progress(), matrix.Current()))
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

		fmt.Println(i18n.T("\nReturn to camera selection"))
	}
}

// experimentMeta describes the current capture conditions for an experiment report.
func experimentMeta(cam *camera.Camera, f pipeline.Frame, cameraName string, fps float64, format, requested string, autoLocked bool) experiment.Meta {
	return experiment.Meta{
		Camera:     cameraName,
		Backend:    cam.Backend(),
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
			steps = append(steps, experiment.Step{Width: m.Width, Height: m.Height, Format: m.Format})
			continue
		}
		for _, fps := range m.FPS {
			steps = append(steps, experiment.Step{Width: m.Width, Height: m.Height, FPS: fps, Format: m.Format})
		}
	}
	return steps
}

// applyStep switches the camera config to a matrix step.
func applyStep(cfg *camera.Config, step experiment.Step) {
	cfg.Width = step.Width
	cfg.Height = step.Height
	cfg.FPS = step.FPS
	cfg.Format = step.Format
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
			matrixText += " " + i18n.T("warming up")
		}
	}

	return ui.ExperimentInfo{
		Mode:      exp.Config.Mode.Title(),
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
		return i18n.Tf("Noise graph closed, data: %s", path)
	}

	opened, err := graph.Open()
	if err != nil {
		slog.Error("noise graph open", "err", err)
		return i18n.Tf("Noise graph error: %v", err)
	}
	*g = opened
	fmt.Print(i18n.Tf("\nNoise graph logging to %s\n", opened.Path()))
	return i18n.Tf("Noise graph opened, logging to %s", opened.Path())
}

// toggleAutoLock locks or restores the camera's automatic exposure, white balance and focus.
// Returns a message for the on-screen notification.
func toggleAutoLock(cam *camera.Camera, locked *bool, saved *camera.Controls) string {
	if *locked {
		cam.UnlockAuto(*saved)
		*locked = false
		slog.Info("auto controls restored", "controls", cam.ReadControls().String())
		return i18n.T("Auto controls restored")
	}

	before, after := cam.LockAuto()
	backend := cam.Backend()
	slog.Info("auto controls lock", "backend", backend, "before", before.String(), "after", after.String())

	if !before.Reported() && !after.Reported() {
		fmt.Print(i18n.Tf("\nAuto controls lock: the %s backend does not give access to exposure, white balance and focus\n", backend))
		if runtime.GOOS == "windows" {
			fmt.Println(i18n.T("Restart the program and choose the DirectShow backend, or set the camera manually with [O]"))
		}
		return i18n.Tf("Auto controls not supported by %s backend", backend)
	}

	applied := camera.LockApplied(before, after)
	fixed := camera.ValuesFixed(before)
	if len(applied) == 0 && len(fixed) == 0 {
		return i18n.T("Auto controls lock ignored by the camera driver")
	}

	*saved = before
	*locked = true
	fmt.Print(i18n.Tf("\nAuto controls locked (%s)\n  before: %s\n  after:  %s\n", backend, before, after))
	if len(applied) > 0 {
		return i18n.Tf("Locked: %s", strings.Join(applied, ", "))
	}
	return i18n.Tf("Fixed at current values: %s (driver does not confirm)", strings.Join(fixed, ", "))
}

// openDriverSettings opens the camera driver's settings dialog, where auto exposure,
// white balance and focus can be switched off by hand. Only DirectShow has the dialog.
func openDriverSettings(cam *camera.Camera) string {
	backend := cam.Backend()
	if backend != camera.BackendName(camera.APIDShow) {
		return i18n.Tf("Driver settings dialog needs the DirectShow backend (now %s)", backend)
	}
	cam.OpenDriverSettings()
	slog.Info("driver settings dialog opened")
	return i18n.T("Driver settings opened: switch off Auto there")
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
