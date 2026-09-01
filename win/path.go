package win

import (
	"syscall"
	"unsafe"
)

var (
	procGetLongPathNameW = kernel32.NewProc("GetLongPathNameW")
	procSearchPathW      = kernel32.NewProc("SearchPathW")
)

// LongPath раскрывает короткие DOS-имена (PROGRA~1) в полные пути.
// Не удалось — возвращает исходную строку.
func LongPath(p string) string {
	ptr, err := syscall.UTF16PtrFromString(p)
	if err != nil {
		return p
	}
	buf := make([]uint16, 1024)
	n, _, _ := procGetLongPathNameW.Call(
		uintptr(unsafe.Pointer(ptr)), uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
	if n == 0 || n >= uintptr(len(buf)) {
		return p
	}
	return syscall.UTF16ToString(buf[:n])
}

// SearchPath ищет исполняемый файл по путям поиска Windows
// (System32 и др.) и возвращает полный путь. Не нашёл — исходную строку.
func SearchPath(file string) string {
	ptr, err := syscall.UTF16PtrFromString(file)
	if err != nil {
		return file
	}
	buf := make([]uint16, 1024)
	n, _, _ := procSearchPathW.Call(0, uintptr(unsafe.Pointer(ptr)), 0,
		uintptr(len(buf)), uintptr(unsafe.Pointer(&buf[0])), 0)
	if n == 0 || n >= uintptr(len(buf)) {
		return file
	}
	return syscall.UTF16ToString(buf[:n])
}
