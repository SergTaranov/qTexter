package win

import (
	"syscall"
	"time"
	"unsafe"
)

// WaitForWindowClass ждёт появления окна заданного класса
// (например "OpusApp" у Word, "Notepad" у Блокнота) не дольше timeout.
// Нужно, чтобы Ctrl+V уйти в ещё не открытое приложение.
func WaitForWindowClass(class string, timeout time.Duration) bool {
	cp, err := syscall.UTF16PtrFromString(class)
	if err != nil {
		return false
	}
	deadline := time.Now().Add(timeout)
	for {
		h, _, _ := procFindWindowW.Call(uintptr(unsafe.Pointer(cp)), 0)
		if h != 0 {
			return true
		}
		if !time.Now().Before(deadline) {
			return false
		}
		time.Sleep(200 * time.Millisecond)
	}
}
