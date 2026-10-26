//go:build windows

package i18n

import "syscall"

// utf8CodePage is the Windows code page for UTF-8.
const utf8CodePage = 65001

// The Windows console uses an OEM code page by default, which shows Cyrillic UTF-8 output as garbage.
func init() {
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	kernel32.NewProc("SetConsoleOutputCP").Call(utf8CodePage)
	kernel32.NewProc("SetConsoleCP").Call(utf8CodePage)
}
