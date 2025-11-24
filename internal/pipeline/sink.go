package pipeline

import (
	"gocv.io/x/gocv"
)

// WindowSink — outputs frames to the window (without waitKey).
func WindowSink(win *gocv.Window) func(Frame) {
	return func(f Frame) {
		win.IMShow(f.Img)
	}
}
