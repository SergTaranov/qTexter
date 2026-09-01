package win

import (
	"fmt"
	"syscall"
	"time"
	"unsafe"
)

const (
	cfUnicodeText = 13 // CF_UNICODETEXT
	gmemMoveable  = 0x0002
)

var (
	procOpenClipboard    = user32.NewProc("OpenClipboard")
	procEmptyClipboard   = user32.NewProc("EmptyClipboard")
	procCloseClipboard   = user32.NewProc("CloseClipboard")
	procSetClipboardData = user32.NewProc("SetClipboardData")
	procRegisterFormat   = user32.NewProc("RegisterClipboardFormatW")
	procGlobalAlloc      = kernel32.NewProc("GlobalAlloc")
	procGlobalLock       = kernel32.NewProc("GlobalLock")
	procGlobalUnlock     = kernel32.NewProc("GlobalUnlock")
	procRtlMoveMemory    = kernel32.NewProc("RtlMoveMemory")
)

// SetClipboardText кладёт текст в системный буфер обмена (CF_UNICODETEXT).
// Буфер бывает занят другим процессом, поэтому открытие повторяется.
func SetClipboardText(text string) error {
	return withOpenClipboard(func() error {
		procEmptyClipboard.Call()
		return setClipboardString(text, cfUnicodeText)
	})
}

// SetClipboardTextAndRTF кладёт текст и, дополнительно, RTF-представление.
// Word/Outlook при вставке предпочитают RTF и сохраняют начертания,
// Блокнот и прочие берут обычный текст.
func SetClipboardTextAndRTF(text, rtf string) error {
	return withOpenClipboard(func() error {
		procEmptyClipboard.Call()
		if err := setClipboardString(text, cfUnicodeText); err != nil {
			return err
		}
		if rtf == "" {
			return nil
		}
		name, err := syscall.UTF16PtrFromString("Rich Text Format")
		if err != nil {
			return nil // RTF не критичен
		}
		fmt_, _, _ := procRegisterFormat.Call(uintptr(unsafe.Pointer(name)))
		if fmt_ == 0 {
			return nil
		}
		// формат RTF — однобайтовые ANSI-данные (не UTF-16!):
		// в UTF-16 Word видит невалидный RTF и вставляет его как текст
		setClipboardBytes(rtf, fmt_)
		return nil
	})
}

// setClipboardString пишет UTF-16 строку в буфер в заданном формате.
// Вызывать только между OpenClipboard и CloseClipboard.
func setClipboardString(s string, format uintptr) error {
	utf16s, err := syscall.UTF16FromString(s)
	if err != nil {
		return err
	}
	size := uintptr(len(utf16s)) * unsafe.Sizeof(utf16s[0])
	h, _, _ := procGlobalAlloc.Call(gmemMoveable, size)
	if h == 0 {
		return fmt.Errorf("GlobalAlloc: не удалось выделить память")
	}
	p, _, _ := procGlobalLock.Call(h)
	if p == 0 {
		return fmt.Errorf("GlobalLock: не удалось заблокировать память")
	}
	// копируем в глобальную память через Win32, а не через unsafe.Pointer(p):
	// обратное преобразование uintptr -> unsafe.Pointer запрещено go vet
	procRtlMoveMemory.Call(p, uintptr(unsafe.Pointer(&utf16s[0])), size)
	procGlobalUnlock.Call(h)

	r, _, _ := procSetClipboardData.Call(format, h)
	if r == 0 {
		return fmt.Errorf("SetClipboardData: ошибка записи в буфер обмена")
	}
	return nil
}

// setClipboardBytes пишет однобайтовые данные (для RTF и подобных
// ANSI-форматов буфера обмена).
func setClipboardBytes(s string, format uintptr) {
	b := []byte(s)
	size := uintptr(len(b)) + 1 // + NUL-терминатор
	h, _, _ := procGlobalAlloc.Call(gmemMoveable, size)
	if h == 0 {
		return
	}
	p, _, _ := procGlobalLock.Call(h)
	if p == 0 {
		return
	}
	procRtlMoveMemory.Call(p, uintptr(unsafe.Pointer(&b[0])), uintptr(len(b)))
	// NUL-терминатор после строки
	var zero byte
	procRtlMoveMemory.Call(p+uintptr(len(b)), uintptr(unsafe.Pointer(&zero)), 1)
	procGlobalUnlock.Call(h)
	procSetClipboardData.Call(format, h)
}

func withOpenClipboard(fn func() error) error {
	opened := false
	for attempt := 0; attempt < 10 && !opened; attempt++ {
		r, _, _ := procOpenClipboard.Call(0)
		opened = r != 0
		if !opened {
			time.Sleep(25 * time.Millisecond)
		}
	}
	if !opened {
		return fmt.Errorf("OpenClipboard: буфер обмена занят другим процессом")
	}
	defer procCloseClipboard.Call()
	return fn()
}
