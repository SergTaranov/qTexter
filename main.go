package main

import (
	"fmt"
	"image/color"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"shorttxt/win"
)

const (
	windowTitle         = "shortTxt3"
	defaultWindowHeight = 460
)

func main() {
	defer logPanic("main")
	a := app.New()
	w := a.NewWindow(windowTitle)
	w.SetIcon(appIcon)

	entry := widget.NewMultiLineEntry()
	entry.Wrapping = fyne.TextWrapWord
	entry.TextStyle = fyne.TextStyle{Monospace: true}

	// кэш последнего выделения: к моменту нажатия кнопки панели
	// Fyne может сбросить выделение поля — помним его по движению курсора
	sel := &selectionCache{entry: entry}
	entry.OnCursorChanged = sel.track
	sel.track() // начальное состояние

	status := widget.NewLabel("")
	updateStatus := func() {
		runes := len([]rune(entry.Text))
		lines := strings.Count(entry.Text, "\n") + 1
		status.SetText(fmt.Sprintf("%d симв. · %d строк", runes, lines))
	}

	history := newHistory(entry, updateStatus)
	entry.OnChanged = func(string) { sel.invalidate(); history.push(); updateStatus() }
	updateStatus()

	// подсказки при наведении на значки: слой поверх содержимого
	tipsUI := newTips(w.Canvas())

	bottom := container.NewVBox(
		container.NewHBox(status),
		newAppBar(a, w, entry, tipsUI),
	)

	// настройки: сохранённые размер шрифта поля и размер окна
	st := LoadSettings()
	SetSize(st.FontSize)

	// сохранение настроек: размер шрифта (по клику кнопки панели) и
	// размер окна. Запись размера окна дебаунсится — драйвер вызывает
	// Resize корня контента при каждом изменении размера окна.
	var winSizeMu sync.Mutex
	savedWin := fyne.NewSize(st.WinW, st.WinH)
	var winSizeTimer *time.Timer
	saveSettings := func() {
		cs := w.Canvas().Size()
		if cs.Width <= 0 || cs.Height <= 0 {
			return // окно ещё не показано
		}
		winSizeMu.Lock()
		defer winSizeMu.Unlock()
		if SaveSettings(appSettings{FontSize: appState.size, WinW: cs.Width, WinH: cs.Height}) == nil {
			savedWin = cs
		}
	}
	onWinResized := func(sz fyne.Size) {
		if sz.Width <= 0 || sz.Height <= 0 {
			return
		}
		winSizeMu.Lock()
		defer winSizeMu.Unlock()
		if sz.Width == savedWin.Width && sz.Height == savedWin.Height {
			return // уже сохранено (в т.ч. программный Resize на старте)
		}
		if winSizeTimer != nil {
			winSizeTimer.Stop()
		}
		winSizeTimer = time.AfterFunc(time.Second, saveSettings)
	}

	toolbar, toolbarMinWidth := newToolbar(a, w, entry, history, sel, saveSettings)

	// поле живёт в собственной теме (шрифт/размер/чёрный цвет),
	// остальной интерфейс — в стандартной: смена размера шрифта
	// поля не двигает панели и надписи
	field := container.NewThemeOverride(entry, entryThemeInstance)

	content := container.NewBorder(toolbar, bottom, nil, nil, field)
	// слой подсказок поверх содержимого: он из неинтерактивных
	// объектов и не мешает кликам по тому, что под ним
	root := container.NewStack(content, tipsUI.object())
	// минимальная ширина окна = ширина панели инструментов:
	// драйвер ограничивает окно по MinSize контента
	wrapped := newMinWidthWrapper(root, toolbarMinWidth, 0, onWinResized)
	w.SetContent(wrapped)
	if st.WinW > 0 && st.WinH > 0 {
		w.Resize(fyne.NewSize(st.WinW, st.WinH)) // размер, сохранённый пользователем
	} else {
		w.Resize(fyne.NewSize(toolbarMinWidth, defaultWindowHeight))
	}
	w.ShowAndRun()
}

// logPanic пишет стек паники в файл рядом с exe — иначе падение
// GUI-приложения остаётся для пользователя немым.
func logPanic(where string) {
	if r := recover(); r != nil {
		msg := fmt.Sprintf("=== %s %s ===\n%s\n\n", time.Now().Format("2006-01-02 15:04:05"), where, debug.Stack())
		dir := os.TempDir()
		if exe, err := os.Executable(); err == nil {
			dir = filepath.Dir(exe)
		}
		f, err := os.OpenFile(filepath.Join(dir, "shorttxt_panic.log"),
			os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
		if err == nil {
			f.WriteString(msg)
			f.Close()
		}
		panic(r) // после записи стек отдаём драйверу
	}
}

// selectionCache помнит последний непустой SelectedText и позицию его
// конца — пока курсор не двигался после выделения (клики по панели
// курсор не двигают), выделение считается живым.
type selectionCache struct {
	entry          *widget.Entry
	text           string
	endRow, endCol int
	valid          bool
}

func (c *selectionCache) track() {
	if c.entry.SelectedText() != "" {
		c.text = c.entry.SelectedText()
		c.endRow, c.endCol = c.entry.CursorRow, c.entry.CursorColumn
		c.valid = true
	} else {
		c.valid = false
	}
}

func (c *selectionCache) invalidate() { c.valid = false }

// undoHistory — простая история изменений для кнопки «Отмена»
// (у Fyne Entry нет публичного undo).
type undoHistory struct {
	mu        *bool // true, пока изменение вызвано самим откатом
	stack     *[]string
	entry     *widget.Entry
	onUndoSet func()
}

func newHistory(entry *widget.Entry, onUndoSet func()) *undoHistory {
	flag := false
	stack := []string{}
	return &undoHistory{mu: &flag, stack: &stack, entry: entry, onUndoSet: onUndoSet}
}

func (h *undoHistory) push() {
	if *h.mu {
		*h.mu = false
		return
	}
	if len(*h.stack) > 200 {
		*h.stack = (*h.stack)[1:]
	}
	*h.stack = append(*h.stack, h.entry.Text)
}

func (h *undoHistory) undo() {
	n := len(*h.stack)
	if n < 2 {
		return
	}
	*h.stack = (*h.stack)[:n-1]
	prev := (*h.stack)[n-2]
	*h.mu = true
	h.entry.SetText(prev)
	if h.onUndoSet != nil {
		h.onUndoSet()
	}
}

// selectionRange — границы текущего выделения в рунах entry.Text.
// У Fyne Entry нет публичных индексов выделения, поэтому: текст берём
// из SelectedText (или из кэша, если поле сбросило выделение), а конец
// выделения — из позиции курсора; из всех вхождений выбираем то, чей
// конец ближе к курсору.
func selectionRange(entry *widget.Entry, cache *selectionCache) (start, end int, ok bool) {
	selText := entry.SelectedText()
	endRow, endCol := entry.CursorRow, entry.CursorColumn
	if selText == "" && cache.valid {
		selText = cache.text
		endRow, endCol = cache.endRow, cache.endCol
	}
	if selText == "" {
		return 0, 0, false
	}
	selRunes := []rune(selText)
	full := []rune(entry.Text)
	cursor := runeOffsetAt(entry, endRow, endCol)

	best, bestDist := -1, int(^uint(0)>>1)
	for i := 0; i+len(selRunes) <= len(full); i++ {
		if string(full[i:i+len(selRunes)]) == selText {
			d := cursor - (i + len(selRunes))
			if d < 0 {
				d = -d
			}
			if d < bestDist {
				best, bestDist = i, d
			}
		}
	}
	if best < 0 {
		return 0, 0, false
	}
	return best, best + len(selRunes), true
}

// replaceRange меняет фрагмент [start,end) на repl, сохраняя позицию
// курсора (иначе он прыгает в начало текста).
func replaceRange(entry *widget.Entry, start, end int, repl string) {
	old := []rune(entry.Text)
	cursor := runeOffsetAt(entry, entry.CursorRow, entry.CursorColumn)
	entry.SetText(string(old[:start]) + repl + string(old[end:]))
	// курсор: до замены — на месте; внутри/после — на конец замены
	switch {
	case cursor <= start:
		restoreCursor(entry, cursor)
	case cursor <= end:
		restoreCursor(entry, start+len([]rune(repl)))
	default:
		restoreCursor(entry, cursor-len([]rune(old[start:end]))+len([]rune(repl)))
	}
}

func restoreCursor(entry *widget.Entry, offset int) {
	r := []rune(entry.Text)
	if offset < 0 {
		offset = 0
	}
	if offset > len(r) {
		offset = len(r)
	}
	row, col := 0, offset
	for i, ch := range r {
		if i >= offset {
			break
		}
		if ch == '\n' {
			row++
			col = offset - i - 1
		}
	}
	entry.CursorRow, entry.CursorColumn = row, col
	entry.Refresh()
}

// minWidthWrapper задаёт минимальный размер содержимому:
// драйвер окна ограничивает размер по MinSize контента. Resize
// вызывается драйвером при каждом изменении размера окна — через
// onResized main узнаёт фактический размер для сохранения.
type minWidthWrapper struct {
	widget.BaseWidget
	inner     fyne.CanvasObject
	min       fyne.Size
	onResized func(fyne.Size)
}

func newMinWidthWrapper(inner fyne.CanvasObject, minW, minH float32, onResized func(fyne.Size)) *minWidthWrapper {
	m := &minWidthWrapper{inner: inner, min: fyne.NewSize(minW, minH), onResized: onResized}
	m.ExtendBaseWidget(m)
	return m
}

func (m *minWidthWrapper) Resize(size fyne.Size) {
	m.BaseWidget.Resize(size)
	if m.onResized != nil {
		m.onResized(size)
	}
}

func (m *minWidthWrapper) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(m.inner)
}

func (m *minWidthWrapper) MinSize() fyne.Size {
	base := m.inner.MinSize()
	return fyne.NewSize(
		fyne.Max(base.Width, m.min.Width),
		fyne.Max(base.Height, m.min.Height),
	)
}

// newToolbar — верхняя панель: буфер обмена, нумерованный список,
// кнопки размера шрифта и меню «Действия».
// Возвращает также минимальную ширину панели для ограничения окна.
// onSettingsChange вызывается после смены размера шрифта — наружу
// сохранение настроек (размер шрифта + текущий размер окна).
func newToolbar(a fyne.App, w fyne.Window, entry *widget.Entry, history *undoHistory, sel *selectionCache, onSettingsChange func()) (fyne.CanvasObject, float32) {
	ib := func(icon fyne.Resource, tip string, fn func()) *widget.Button {
		b := widget.NewButtonWithIcon("", icon, fn)
		b.Importance = widget.LowImportance
		return b
	}
	refocus := func() { w.Canvas().Focus(entry) }

	// RTF-представление выделения или всего текста: Word/Outlook
	// вставляют его с начертаниями
	clipRTF := func() (text string, rtf string) {
		if s, e, ok := selectionRange(entry, sel); ok {
			text = string([]rune(entry.Text)[s:e])
		} else {
			text = entry.Text
		}
		return text, buildRTF(text, entry.TextStyle)
	}

	cutBtn := ib(theme.ContentCutIcon(), "Вырезать", func() {
		text, rtf := clipRTF()
		win.SetClipboardTextAndRTF(text, rtf)
		if s, e, ok := selectionRange(entry, sel); ok {
			replaceRange(entry, s, e, "")
			refocus()
			return
		}
		entry.SetText("")
		refocus()
	})
	copyBtn := ib(theme.ContentCopyIcon(), "Копировать", func() {
		text, rtf := clipRTF()
		win.SetClipboardTextAndRTF(text, rtf)
	})
	pasteBtn := ib(theme.ContentPasteIcon(), "Вставить", func() {
		clip := a.Clipboard().Content()
		if clip == "" {
			return
		}
		if s, e, ok := selectionRange(entry, sel); ok {
			replaceRange(entry, s, e, clip)
			refocus()
			return
		}
		pos := runeOffsetAt(entry, entry.CursorRow, entry.CursorColumn)
		r := []rune(entry.Text)
		if pos > len(r) {
			pos = len(r)
		}
		entry.SetText(string(r[:pos]) + clip + string(r[pos:]))
		refocus()
	})
	undoBtn := ib(theme.ContentUndoIcon(), "Отмена", func() {
		history.undo()
		refocus()
	})

	// --- нумерованный список ---
	listBtn := ib(formatIconResource("numlist"), "Список с цифрами", func() {
		applyOp(entry, sel, NumberedList)
		refocus()
	})

	// --- размер шрифта: «−», текущее значение (клик — список), «+» ---
	var sizeValue *sizeValueLabel
	applySize := func(v float32) {
		if v < minFontSize || v > maxFontSize || v == appState.size {
			return // предел диапазона достигнут
		}
		SetSize(v)
		sizeValue.SetText(fmt.Sprintf("%.0f", appState.size))
		rerender(entry)
		onSettingsChange()
		refocus()
	}
	sizeValue = newSizeValueLabel(fmt.Sprintf("%.0f", appState.size), func() {
		// клик по цифре — сетка размеров. Не PopUpMenu: 21 пункт выше
		// окна, Fyne делает список прокручиваемым, и оверлейный скроллбар
		// при наведении перекрывает цифры. Сетка 4×6 помещается целиком.
		var pop *widget.PopUp
		btns := make([]fyne.CanvasObject, 0, maxFontSize-minFontSize+1)
		for s := minFontSize; s <= maxFontSize; s++ {
			s := s
			b := widget.NewButton(fmt.Sprintf("%d", s), func() {
				pop.Hide()
				applySize(float32(s))
			})
			if float32(s) == appState.size {
				b.Importance = widget.HighImportance // текущий размер
			}
			btns = append(btns, b)
		}
		bg := canvas.NewRectangle(theme.Color(theme.ColorNameBackground))
		bg.CornerRadius = 4
		frame := canvas.NewRectangle(color.Transparent)
		frame.StrokeColor = theme.Color(theme.ColorNameForeground)
		frame.StrokeWidth = 1
		frame.CornerRadius = 4
		grid := container.NewGridWithColumns(4, btns...)
		body := container.NewStack(bg, frame, container.NewPadded(grid))
		pop = widget.NewPopUp(body, w.Canvas())
		pos := fyne.CurrentApp().Driver().AbsolutePositionForObject(sizeValue)
		pop.ShowAtPosition(pos.Add(fyne.NewPos(0, sizeValue.Size().Height)))
	})
	sizeBox := container.NewGridWrap(fyne.NewSize(32, 34), sizeValue)
	sizeDownBtn := ib(theme.ContentRemoveIcon(), "Уменьшить шрифт", func() { applySize(appState.size - 1) })
	sizeUpBtn := ib(theme.ContentAddIcon(), "Увеличить шрифт", func() { applySize(appState.size + 1) })

	// «Действия» — только три точки, подпись не нужна
	actionsBtn := widget.NewButtonWithIcon("", theme.MoreVerticalIcon(), nil)
	actionsBtn.Importance = widget.LowImportance
	actionsBtn.OnTapped = func() {
		pos := fyne.CurrentApp().Driver().AbsolutePositionForObject(actionsBtn)
		items := make([]*fyne.MenuItem, 0, len(toolbarOps)+2)
		for _, o := range toolbarOps {
			o := o
			items = append(items, fyne.NewMenuItem(o.label, func() {
				applyOp(entry, sel, o.fn)
				refocus()
			}))
		}
		items = append(items, fyne.NewMenuItemSeparator(),
			fyne.NewMenuItem("Копировать всё", func() {
				win.SetClipboardTextAndRTF(entry.Text, buildRTF(entry.Text, entry.TextStyle))
			}),
			fyne.NewMenuItem("Очистить", func() {
				entry.SetText("")
				refocus()
			}))
		pop := widget.NewPopUpMenu(fyne.NewMenu("Действия", items...), w.Canvas())
		pop.ShowAtPosition(pos.Add(fyne.NewPos(0, actionsBtn.Size().Height)))
	}

	row := container.NewHBox(
		cutBtn, copyBtn, pasteBtn, undoBtn,
		widget.NewSeparator(),
		listBtn,
		widget.NewSeparator(),
		sizeDownBtn, sizeBox, sizeUpBtn,
		widget.NewSeparator(),
		actionsBtn,
	)
	minW := row.MinSize().Width + theme.Padding()*6 + 16 // + скроллбар и рамки
	return container.NewHScroll(row), minW
}

// sizeValueLabel — цифра размера шрифта; по клику открывает список
// размеров. Отдельный тип нужен, чтобы Label получил нажатие
// и курсор-руку.
type sizeValueLabel struct {
	widget.Label
	onTap func()
}

func newSizeValueLabel(text string, onTap func()) *sizeValueLabel {
	l := &sizeValueLabel{onTap: onTap}
	l.ExtendBaseWidget(l)
	l.SetText(text)
	return l
}

func (l *sizeValueLabel) Tapped(*fyne.PointEvent) {
	if l.onTap != nil {
		l.onTap()
	}
}

// Cursor — курсор-рука над цифрой, чтобы клик был очевиден.
func (l *sizeValueLabel) Cursor() desktop.Cursor { return desktop.PointerCursor }

// rerender — полный пересбор отрисовки поля при смене размера
// шрифта. Простого Refresh недостаточно: Entry
// перерисовывает с новым шрифтом только строку с курсором, остальные
// строки остаются со старым. Кратковременное выключение/включение
// переноса заставляет пересоздать все строковые объекты.
func rerender(entry *widget.Entry) {
	entry.Wrapping = fyne.TextWrapOff
	entry.Refresh()
	entry.Wrapping = fyne.TextWrapWord
	entry.Refresh()
	fyne.Do(func() { entry.Refresh() })
}

// runeOffsetAt — смещение (строка, колонка) в рунах текста entry.
func runeOffsetAt(entry *widget.Entry, row, col int) int {
	lines := strings.Split(entry.Text, "\n")
	off := 0
	for i := 0; i < row && i < len(lines); i++ {
		off += len([]rune(lines[i])) + 1
	}
	return off + col
}

// op — текстовая операция для меню «Действия».
type op struct {
	label string
	fn    func(string) string
}

var toolbarOps = []op{
	{"ВЕРХНИЙ РЕГИСТР", UpperCase},
	{"нижний регистр", LowerCase},
	{"Как в предложении", SentenceCase},
	{"Обрезать пробелы", TrimLines},
	{"В одну строку", JoinLines},
	{"Убрать пустые строки", RemoveEmptyLines},
	{"Сортировать", SortLines},
	{"Уникальные", UniqueLines},
	{"Реверс", Reverse},
}

// applyOp применяет операцию к выделенному фрагменту (с учётом кэша
// выделения), иначе ко всему тексту.
func applyOp(entry *widget.Entry, sel *selectionCache, fn func(string) string) {
	if s, e, ok := selectionRange(entry, sel); ok {
		replaceRange(entry, s, e, fn(string([]rune(entry.Text)[s:e])))
		return
	}
	replaceRange(entry, 0, len([]rune(entry.Text)), fn(entry.Text))
}

// newAppBar — нижняя панель значков приложений (без подписей; имя
// приложения показывается подсказкой при наведении).
// F5 перечитывает apps.json без перезапуска.
func newAppBar(a fyne.App, w fyne.Window, entry *widget.Entry, tipsUI *tips) fyne.CanvasObject {
	var wrap *fyne.Container

	fill := func() {
		apps, err := LoadApps()
		if err != nil {
			fyne.Do(func() {
				dialog.ShowError(fmt.Errorf("не удалось загрузить apps.json: %v", err), w)
			})
			return
		}
		box := container.NewHBox()
		for _, t := range apps {
			t := t
			btn := newTipButton(targetResource(t), t.Name, func() {
				// читаем текст и стили в потоке UI, RTF собираем здесь же
				text := entry.Text
				rtf := buildRTF(text, entry.TextStyle)
				go func() {
					win.MinimizeWindowByTitle(windowTitle)
					if err := Launch(t, text, rtf); err != nil {
						fyne.Do(func() { dialog.ShowError(err, w) })
					}
				}()
			}, tipsUI)
			box.Add(btn)
		}
		wrap.RemoveAll()
		wrap.Add(container.NewHScroll(box))
		wrap.Refresh()
	}

	wrap = container.NewMax()
	fill()

	w.Canvas().AddShortcut(&desktop.CustomShortcut{KeyName: fyne.KeyF5},
		func(fyne.Shortcut) { fill() })
	return wrap
}
