//go:build darwin

package keyboard

import "gocv.io/x/gocv"

const (
	keyArrowUp    = 0
	keyArrowDown  = 1
	keyArrowLeft  = 2
	keyArrowRight = 3
)

func readKey(win *gocv.Window) int {
	return win.WaitKey(1)
}
