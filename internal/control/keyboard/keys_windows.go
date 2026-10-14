//go:build windows

package keyboard

import (
	"syscall"
	"unsafe"

	"gocv.io/x/gocv"
)

// Arrow codes are only labels now; they do not come from OpenCV on Windows.
const (
	keyArrowUp    = 2490368
	keyArrowDown  = 2621440
	keyArrowLeft  = 2424832
	keyArrowRight = 2555904
)

// Must match the name passed to gocv.NewWindow in main.go.
const windowTitle = "Camera"

const (
	vkLeft  = 0x25
	vkUp    = 0x26
	vkRight = 0x27
	vkDown  = 0x28
)

var (
	user32                  = syscall.NewLazyDLL("user32.dll")
	procGetAsyncKeyState    = user32.NewProc("GetAsyncKeyState")
	procGetForegroundWindow = user32.NewProc("GetForegroundWindow")
	procFindWindowW         = user32.NewProc("FindWindowW")
)

var arrows = []struct {
	vk   uintptr
	code int
	down bool
}{
	{vkUp, keyArrowUp, false},
	{vkDown, keyArrowDown, false},
	{vkLeft, keyArrowLeft, false},
	{vkRight, keyArrowRight, false},
}

// cameraWindowFocused reports whether the "Camera" window is in the foreground,
// so arrows pressed in other programs are ignored.
func cameraWindowFocused() bool {
	title, err := syscall.UTF16PtrFromString(windowTitle)
	if err != nil {
		return false
	}
	hwnd, _, _ := procFindWindowW.Call(0, uintptr(unsafe.Pointer(title)))
	fg, _, _ := procGetForegroundWindow.Call()
	return hwnd != 0 && hwnd == fg
}

// pollArrow returns an arrow code once per key press, or 0 if none was pressed.
func pollArrow() int {
	focused := cameraWindowFocused()
	result := 0

	for i := range arrows {
		state, _, _ := procGetAsyncKeyState.Call(arrows[i].vk)
		isDown := state&0x8000 != 0

		if isDown && !arrows[i].down && focused && result == 0 {
			result = arrows[i].code
		}
		arrows[i].down = isDown
	}

	return result
}

// readKey pumps window messages via WaitKey (letters, digits, ESC, TAB arrive
// normally) and reads the arrow keys directly from Windows.
func readKey(win *gocv.Window) int {
	key := win.WaitKey(1)
	if arrow := pollArrow(); arrow != 0 {
		return arrow
	}
	return key
}
