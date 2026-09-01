package main

import (
	"image/color"
	"os"
	"path/filepath"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

// Шрифт и размеры по умолчанию как в Lister Total Commander:
// фиксированный шрифт (Lucida Console), без выбора.
const (
	defaultFontFile = "lucon.ttf" // Lucida Console
	defaultFontSize = 12
	minFontSize     = 8
	maxFontSize     = 28
)

// appState — настройки внешнего вида текстового поля: фиксированный
// моноширинный шрифт и размер, меняемый кнопками панели.
// fontName нужен для RTF в буфере обмена (buildRTF).
var appState = struct {
	fontRes  fyne.Resource
	fontName string
	size     float32
}{
	size: defaultFontSize,
}

func init() {
	path := filepath.Join(os.Getenv("WINDIR"), "Fonts", defaultFontFile)
	if data, err := os.ReadFile(path); err == nil {
		appState.fontRes = fyne.NewStaticResource(defaultFontFile, data)
		appState.fontName = "Lucida Console"
	}
}

// entryTheme — тема ТОЛЬКО текстового поля (применяется через
// container.NewThemeOverride): свой моноширинный шрифт, свой размер
// текста и чёрный цвет «как в Блокноте». Остальной интерфейс живёт
// в стандартной теме и не реагирует на смену размера шрифта поля.
type entryTheme struct {
	fyne.Theme
}

// entryThemeInstance передаётся в NewThemeOverride один раз; методы
// читают appState при каждой отрисовке, поэтому смена размера
// подхватывается без пересоздания контейнера.
var entryThemeInstance = &entryTheme{Theme: theme.DefaultTheme()}

func (t *entryTheme) Font(s fyne.TextStyle) fyne.Resource {
	if s.Monospace && appState.fontRes != nil {
		return appState.fontRes
	}
	return t.Theme.Font(s)
}

func (t *entryTheme) Color(n fyne.ThemeColorName, v fyne.ThemeVariant) color.Color {
	if n == theme.ColorNameForeground {
		return color.Black // чёрный текст — как в Блокноте
	}
	return t.Theme.Color(n, v)
}

func (t *entryTheme) Size(n fyne.ThemeSizeName) float32 {
	if n == theme.SizeNameText {
		return appState.size
	}
	return t.Theme.Size(n)
}

// SetSize меняет размер текста поля.
func SetSize(size float32) {
	if size < minFontSize {
		size = minFontSize
	}
	if size > maxFontSize {
		size = maxFontSize
	}
	appState.size = size
}
