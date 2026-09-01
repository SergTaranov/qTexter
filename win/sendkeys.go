package win

import (
	"fmt"
	"strings"
	"time"
	"unsafe"
)

const (
	inputKeyboard = 1 // INPUT_KEYBOARD
	keyEventKeyUp = 0x0002
	vkControl     = 0x11
	vkV           = 0x56
)

// vkNames — виртуальные коды клавиш для preKeys конфига apps.json.
var vkNames = map[string]uint16{
	"tab":   0x09,
	"enter": 0x0D,
	"space": 0x20,
	"esc":   0x1B,
	"down":  0x28,
	"up":    0x26,
}

type keyboardInput struct {
	wVk         uint16
	wScan       uint16
	dwFlags     uint32
	time        uint32
	dwExtraInfo uintptr
}

// input повторяет win32-структуру INPUT: поле-объединение имеет размер
// самой большой структуры (MOUSEINPUT = 32 байта на x64), поэтому после
// keyboardInput добиваем паддинг до общего размера 40 байт.
type input struct {
	typ uint32
	ki  keyboardInput
	_   [8]byte
}

var procSendInput = user32.NewProc("SendInput")

// SendCtrlV посылает в активное окно сочетание Ctrl+V (вставка из буфера).
func SendCtrlV() error {
	down := []input{
		{typ: inputKeyboard, ki: keyboardInput{wVk: vkControl}},
		{typ: inputKeyboard, ki: keyboardInput{wVk: vkV}},
	}
	up := []input{
		{typ: inputKeyboard, ki: keyboardInput{wVk: vkV, dwFlags: keyEventKeyUp}},
		{typ: inputKeyboard, ki: keyboardInput{wVk: vkControl, dwFlags: keyEventKeyUp}},
	}
	if err := sendInputs(down); err != nil {
		return err
	}
	// пауза между нажатием и отпусканием: некоторые приложения
	// не различают мгновенно следующие друг за другом события
	time.Sleep(30 * time.Millisecond)
	return sendInputs(up)
}

func sendInputs(inputs []input) error {
	n := uintptr(len(inputs))
	r, _, _ := procSendInput.Call(n, uintptr(unsafe.Pointer(&inputs[0])), unsafe.Sizeof(inputs[0]))
	if r != n {
		return fmt.Errorf("SendInput: отправлено %d из %d событий", r, n)
	}
	return nil
}

// SendNamedKey посылает одиночное нажатие клавиши по имени
// (tab, enter, space, esc, down, up).
func SendNamedKey(name string) error {
	vk, ok := vkNames[strings.ToLower(name)]
	if !ok {
		return fmt.Errorf("неизвестная клавиша %q", name)
	}
	down := []input{{typ: inputKeyboard, ki: keyboardInput{wVk: vk}}}
	up := []input{{typ: inputKeyboard, ki: keyboardInput{wVk: vk, dwFlags: keyEventKeyUp}}}
	if err := sendInputs(down); err != nil {
		return err
	}
	time.Sleep(30 * time.Millisecond)
	return sendInputs(up)
}
