// Package win — тонкие обёртки над Win32 API: буфер обмена, эмуляция
// клавиш, извлечение иконок и управление окном. Только для Windows.
package win

import (
	"syscall"
)

var (
	user32   = syscall.NewLazyDLL("user32.dll")
	kernel32 = syscall.NewLazyDLL("kernel32.dll")
	gdi32    = syscall.NewLazyDLL("gdi32.dll")
	shell32  = syscall.NewLazyDLL("shell32.dll")
)
