// package main

// import (
// 	"fmt"
// 	"log"
// 	"runtime"
// 	"webcam/internal/camera"
// 	"webcam/internal/pipeline"

// 	"gocv.io/x/gocv"
// )

// func main() {
// 	runtime.LockOSThread()

// 	cfg := camera.Config{
// 		Index:  1,
// 		API:    camera.APIAny,
// 		Width:  1280,
// 		Height: 720,
// 		FPS:    30,
// 	}

// 	cam, err := camera.Open(cfg)
// 	if err != nil {
// 		log.Fatalf("camera not opened: %v", err)
// 	}
// 	defer cam.Close()

// 	win := gocv.NewWindow("Pipeline Demo")
// 	defer win.Close()

// 	done := make(chan struct{})

// 	pipe := pipeline.New(
// 		done,
// 		pipeline.GrayStage(),
// 		pipeline.BrightnessContrastStage(1.0, 20.0),
// 		pipeline.ToBGRStage(),
// 	)

// 	frames := pipe.Run(pipeline.ReadStage(cam, done))
// 	sink := pipeline.WindowSink(win)

// 	fmt.Println("Press ESC to exit")

// 	for {
// 		select {
// 		case f, ok := <-frames:
// 			if !ok {
// 				close(done)
// 				return
// 			}
// 			sink(f)

//			default:
//				if win.WaitKey(1) == 27 {
//					close(done)
//					return
//				}
//			}
//		}
//	}
//
// /////////////////////////////////
// /////////////////////////////////
// /////////////////////////////////
// /////////////////////////////////
// /////////////////////////////////
package main

import (
	"log"
	"webcam/internal/camera"
	"webcam/internal/control"
	"webcam/internal/control/keyboard"
	"webcam/internal/pipeline"

	"gocv.io/x/gocv"
)

func main() {
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

	win := gocv.NewWindow("Pipeline Control")
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
			win.IMShow(f.Img)

			cont, changed := keyboard.HandleKeyboard(win, params, stages)
			if !cont {
				close(done)
				return
			}

			if changed {
				break
			}

			// fmt.Printf("alpha=%.2f beta=%.2f blur=%d th1=%.0f th2=%.0f\n",
			// 	params.BrightnessContrast.Alpha,
			// 	params.BrightnessContrast.Beta,
			// 	params.Blur.Ksize,
			// 	params.Edge.Threshold1,
			// 	params.Edge.Threshold2,
			// )
		}
	}
}
