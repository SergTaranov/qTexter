package win

import (
	"fmt"
	"strings"
	"syscall"
	"time"
	"unsafe"
)

const (
	wmSysCommand = 0x0112
	scMinimize   = 0xF020
	swShowNormal = 5
)

var (
	procFindWindowW   = user32.NewProc("FindWindowW")
	procFindWindowExW = user32.NewProc("FindWindowExW")
	procPostMessageW  = user32.NewProc("PostMessageW")
	procShellExecuteW = shell32.NewProc("ShellExecuteW")
	procGetForeground = user32.NewProc("GetForegroundWindow")
	procGetClassNameW = user32.NewProc("GetClassNameW")
	procGetWindowTID  = user32.NewProc("GetWindowThreadProcessId")
)

// WindowProcessID — PID процесса-владельца окна.
func WindowProcessID(h uintptr) uint32 {
	var pid uint32
	procGetWindowTID.Call(h, uintptr(unsafe.Pointer(&pid)))
	return pid
}

// findWizardOfProcess — диалог Office (NUIDialog) в процессе pid или 0.
func findWizardOfProcess(pid uint32) uintptr {
	cp, err := syscall.UTF16PtrFromString("NUIDialog")
	if err != nil {
		return 0
	}
	var prev uintptr
	for {
		h, _, _ := procFindWindowExW.Call(0, prev, uintptr(unsafe.Pointer(cp)), 0)
		if h == 0 {
			return 0
		}
		if WindowProcessID(h) == pid {
			return h
		}
		prev = h
	}
}

// CloseProcessWizard ждёт до timeout появления мастера активации
// Office (NUIDialog) В ПРОЦЕССЕ целевого окна и закрывает его. Мастер
// выскакивает примерно через секунду после старта Word/Outlook и
// съедает вставку, ушедшую раньше него. Возвращает true, если мастер
// был найден и закрыт.
func CloseProcessWizard(targetHwnd uintptr, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for {
		if h := findWizardOfProcess(WindowProcessID(targetHwnd)); h != 0 {
			procPostMessageW.Call(h, wmClose, 0, 0)
			for i := 0; i < 20; i++ { // ждём, пока диалог исчезнет
				if findWizardOfProcess(WindowProcessID(targetHwnd)) == 0 {
					break
				}
				time.Sleep(50 * time.Millisecond)
			}
			return true
		}
		if !time.Now().Before(deadline) {
			return false
		}
		time.Sleep(50 * time.Millisecond)
	}
}

// WindowHandlesByClass возвращает множество дескрипторов окон верхнего
// уровня заданного класса (включая пока невидимые).
func WindowHandlesByClass(class string) map[uintptr]bool {
	cp, err := syscall.UTF16PtrFromString(class)
	if err != nil {
		return nil
	}
	res := map[uintptr]bool{}
	var prev uintptr
	for {
		h, _, _ := procFindWindowExW.Call(0, prev, uintptr(unsafe.Pointer(cp)), 0)
		if h == 0 {
			break
		}
		res[h] = true
		prev = h
	}
	return res
}

// WaitForNewWindowClass ждёт появления НОВОГО окна класса class (не
// входившего в known) и возвращает его дескриптор. Нужно, когда окна
// того же класса уже были открыты: вставлять надо именно в новое.
func WaitForNewWindowClass(class string, known map[uintptr]bool, timeout time.Duration) (uintptr, bool) {
	cp, err := syscall.UTF16PtrFromString(class)
	if err != nil {
		return 0, false
	}
	deadline := time.Now().Add(timeout)
	for {
		var prev uintptr
		for {
			h, _, _ := procFindWindowExW.Call(0, prev, uintptr(unsafe.Pointer(cp)), 0)
			if h == 0 {
				break
			}
			if !known[h] {
				return h, true
			}
			prev = h
		}
		if !time.Now().Before(deadline) {
			return 0, false
		}
		time.Sleep(100 * time.Millisecond)
	}
}

// ForegroundWindow — дескриптор активного окна или 0.
func ForegroundWindow() uintptr {
	h, _, _ := procGetForeground.Call()
	return h
}

// WaitForForegroundHandleStable ждёт, пока окно h непрерывно останется
// активным не меньше stable; пока оно неактивно, периодически вызывает
// dismiss (закрытие диалогов, перехватывающих фокус — например мастера
// активации Office). Разовая проверка недостаточна: диалог может
// появиться спустя секунду после того, как окно цели уже побывало
// активным, и съесть вставку.
func WaitForForegroundHandleStable(h uintptr, stable, timeout time.Duration, dismiss func()) bool {
	deadline := time.Now().Add(timeout)
	var since time.Time
	lastDismiss := time.Now()
	for {
		if ForegroundWindow() == h {
			if since.IsZero() {
				since = time.Now()
			}
			if time.Since(since) >= stable {
				return true
			}
		} else {
			since = time.Time{}
			if dismiss != nil && time.Since(lastDismiss) >= time.Second {
				dismiss()
				lastDismiss = time.Now()
			}
		}
		if !time.Now().Before(deadline) {
			return false
		}
		time.Sleep(100 * time.Millisecond)
	}
}

// FindWindowByTitle возвращает handle окна с указанным заголовком или 0.
func FindWindowByTitle(title string) uintptr {
	ptr, err := syscall.UTF16PtrFromString(title)
	if err != nil {
		return 0
	}
	h, _, _ := procFindWindowW.Call(0, uintptr(unsafe.Pointer(ptr)))
	return h
}

// ForegroundClassName — класс активного окна (например "Notepad").
func ForegroundClassName() string {
	h, _, _ := procGetForeground.Call()
	if h == 0 {
		return ""
	}
	buf := make([]uint16, 256)
	n, _, _ := procGetClassNameW.Call(h, uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
	if n == 0 {
		return ""
	}
	return syscall.UTF16ToString(buf[:n])
}

// WaitForForegroundClass ждёт, пока активное окно получит заданный класс
// (после запуска цели оно появляется с задержкой и не всегда сразу активно).
func WaitForForegroundClass(class string, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for {
		if strings.EqualFold(ForegroundClassName(), class) {
			return true
		}
		if !time.Now().Before(deadline) {
			return false
		}
		time.Sleep(100 * time.Millisecond)
	}
}

const wmClose = 0x0010

// CloseOfficeWizards закрывает всплывающие диалоги Office (класс
// NUIDialog — прежде всего «Мастер активации Microsoft Office»):
// при сбойной активации продукта он появляется при каждом запуске
// Word/Outlook, крадёт фокус, и автоматическая вставка уходит в него.
// Закрытие равнозначно крестику окна.
func CloseOfficeWizards() {
	cp, err := syscall.UTF16PtrFromString("NUIDialog")
	if err != nil {
		return
	}
	for i := 0; i < 4; i++ { // диалогов может быть несколько — по числу приложений
		h, _, _ := procFindWindowW.Call(uintptr(unsafe.Pointer(cp)), 0)
		if h == 0 {
			return
		}
		procPostMessageW.Call(h, wmClose, 0, 0)
		time.Sleep(150 * time.Millisecond)
	}
}

// MinimizeWindowByTitle сворачивает окно с указанным заголовком,
// чтобы фокус и автопаста достались запущенному приложению.
// Работает из любого потока: PostMessage асинхронен.
func MinimizeWindowByTitle(title string) {
	ptr, err := syscall.UTF16PtrFromString(title)
	if err != nil {
		return
	}
	hwnd, _, _ := procFindWindowW.Call(0, uintptr(unsafe.Pointer(ptr)))
	if hwnd == 0 {
		return
	}
	procPostMessageW.Call(hwnd, wmSysCommand, scMinimize, 0)
}

// OpenURL открывает ссылку приложением по умолчанию (обычно браузером).
func OpenURL(rawurl string) error {
	u, err := syscall.UTF16PtrFromString(rawurl)
	if err != nil {
		return err
	}
	verb, _ := syscall.UTF16PtrFromString("open")
	r, _, err := procShellExecuteW.Call(0, uintptr(unsafe.Pointer(verb)),
		uintptr(unsafe.Pointer(u)), 0, 0, swShowNormal)
	if r <= 32 {
		return fmt.Errorf("ShellExecuteW: не удалось открыть %q: %v", rawurl, err)
	}
	return nil
}
