package main

import (
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// tooltipDelay — пауза перед показом подсказки: мгновенно всплывающее
// окно мелькает при простом проведении мыши над рядом значков.
const tooltipDelay = 500 * time.Millisecond

// tips — плавающая подсказка рядом с указателем мыши. Слой состоит
// из неинтерактивных объектов (прямоугольник и текст не реализуют
// Tappable/Hoverable), поэтому подсказка не перехватывает клики и
// наведение у кнопок под ней.
type tips struct {
	canvas fyne.Canvas
	layer  *fyne.Container
	rect   *canvas.Rectangle
	label  *canvas.Text
}

// newTips создаёт контроллер подсказок; object() нужно добавить в
// контейнер поверх основного содержимого (например, в container.NewStack).
func newTips(c fyne.Canvas) *tips {
	t := &tips{
		canvas: c,
		rect: &canvas.Rectangle{
			CornerRadius: 4,
			FillColor:    theme.Color(theme.ColorNameBackground),
			StrokeColor:  theme.Color(theme.ColorNameForeground),
			StrokeWidth:  1,
		},
		label: &canvas.Text{
			TextSize: theme.Size(theme.SizeNameText),
			Color:    theme.Color(theme.ColorNameForeground),
		},
	}
	t.layer = container.NewWithoutLayout(t.rect, t.label)
	t.layer.Hide()
	return t
}

// object возвращает слой подсказок для размещения поверх контента.
func (t *tips) object() fyne.CanvasObject { return t.layer }

// show выводит подсказку text рядом с экранными координатами курсора
// at (в системе координат канвы — как ev.AbsolutePosition).
// Важно: слой сам не двигается — он лежит в стеке на весь размер окна,
// и лейаут стека при перерисовке вернул бы его в (0,0), подсказка
// «уезжала» в угол окна. Двигаются только прямоугольник и текст внутри
// слоя: контейнер слоя — WithoutLayout, позиции его детей не трогает.
func (t *tips) show(text string, at fyne.Position) {
	t.label.Text = text
	t.label.Refresh()
	size := t.label.MinSize().Add(fyne.NewSize(theme.Padding()*3, theme.Padding()*2))

	x := at.X + theme.Padding()*2
	y := at.Y + theme.Padding()*4             // чуть ниже курсора, рядом с указателем
	if x+size.Width > t.canvas.Size().Width { // не вылезать за правый край
		x = t.canvas.Size().Width - size.Width
	}
	if y+size.Height > t.canvas.Size().Height { // не вылезать за нижний край
		y = at.Y - size.Height - theme.Padding()*2
	}
	t.rect.Resize(size)
	t.rect.Move(fyne.NewPos(x, y))
	t.label.Resize(t.label.MinSize())
	t.label.Move(fyne.NewPos(x+theme.Padding()*1.5, y+theme.Padding()))
	t.layer.Show()
}

func (t *tips) hide() { t.layer.Hide() }

// tipButton — кнопка-значок с названием во всплывающей подсказке:
// значки без подписей сами не говорят, куда нажали. Подсказка
// появляется рядом с курсором после паузы tooltipDelay. Обычное
// поведение кнопки сохраняется делегированием во встроенную Button.
type tipButton struct {
	widget.Button
	tip     string
	tips    *tips
	hovered bool
	pos     fyne.Position
	timer   *time.Timer
}

func newTipButton(icon fyne.Resource, tip string, fn func(), t *tips) *tipButton {
	b := &tipButton{tip: tip, tips: t}
	b.Icon = icon
	b.OnTapped = fn
	b.Importance = widget.LowImportance
	b.ExtendBaseWidget(b)
	return b
}

func (b *tipButton) MouseIn(ev *desktop.MouseEvent) {
	b.Button.MouseIn(ev)
	b.hovered = true
	b.pos = ev.AbsolutePosition
	b.timer = time.AfterFunc(tooltipDelay, func() {
		fyne.Do(func() {
			if b.hovered {
				b.tips.show(b.tip, b.pos)
			}
		})
	})
}

// MouseMoved следит за курсором, чтобы подсказка появилась там, где
// указатель остановился, а не где он вошёл на кнопку.
func (b *tipButton) MouseMoved(ev *desktop.MouseEvent) {
	b.Button.MouseMoved(ev)
	b.pos = ev.AbsolutePosition
}

func (b *tipButton) MouseOut() {
	b.Button.MouseOut()
	b.hovered = false
	if b.timer != nil {
		b.timer.Stop()
		b.timer = nil
	}
	b.tips.hide()
}
