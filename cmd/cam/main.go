package main

import (
	"image"
	"image/color"
	"log"
	"runtime"
	"webcam/internal/camera"
	"webcam/internal/control"
	"webcam/internal/control/keyboard"
	"webcam/internal/pipeline"

	"gocv.io/x/gocv"
)

func main() {
	runtime.LockOSThread()

	cams := camera.FindCameras()
	camera.CompareInternalExternal(cams)

	var index int = 0

	for _, cam := range cams {
		if cam.IsExternal {
			index = cam.Index
			break
		}
	}

	cfg := camera.Config{
		Index: index,
		API:   camera.APIAny,
	}

	cam, err := camera.Open(cfg)
	if err != nil {
		log.Fatalf("camera not opened: %v", err)
	}
	defer cam.Close()

	win := gocv.NewWindow("Camera")
	defer win.Close()

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

	done := make(chan struct{})
	for {
		frames := pipeline.BuildPipeline(cam, stages, params, done)

		for f := range frames {
			help := []string{
				"1: Toggle Brightness/Contrast",
				"2: Toggle Blur",
				"3: Toggle Edge",
				"4: Toggle Gray",
				"5: Toggle Sharpen",
				"ESC: Quit",
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
				close(done)
				return
			}

			if changed {
				break
			}
		}
	}
}
