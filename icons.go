package main

import (
	"fmt"
	"path/filepath"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"

	"qtexter/win"
)

// builtinIcons — контурные пиктограммы (белым по цветной подложке),
// рисуются Fyne из SVG, поэтому шрифты не нужны.
var builtinIcons = map[string]string{
	"search": `<circle cx="10.5" cy="10.5" r="6"/>
		<line x1="15" y1="15" x2="20" y2="20"/>`,
	"translate": `<path d="M4 5h10v8H8l-4 4z"/>
		<path d="M11 9h9v7h-5l-3 3v-3h-1"/>
		<line x1="7" y1="7.5" x2="11" y2="7.5"/>
		<line x1="14.5" y1="11.5" x2="18" y2="11.5"/>`,
	"mail": `<rect x="3.5" y="6" width="17" height="12" rx="2"/>
		<path d="M4 8.5l8 5.5 8-5.5"/>`,
	"doc": `<path d="M7 3.5h7l4 4v13H7z"/>
		<path d="M14 3.5v4h4"/>
		<line x1="10" y1="12" x2="15.5" y2="12"/>
		<line x1="10" y1="15" x2="15.5" y2="15"/>`,
	"chat": `<path d="M4 5h16v10H10l-6 4z"/>
		<circle cx="9" cy="10" r="0.6"/>
		<circle cx="12.5" cy="10" r="0.6"/>
		<circle cx="16" cy="10" r="0.6"/>`,
	"globe": `<circle cx="12" cy="12" r="8.5"/>
		<ellipse cx="12" cy="12" rx="4" ry="8.5"/>
		<line x1="3.5" y1="12" x2="20.5" y2="12"/>`,
}

// formatIcons — монохромные значки панели форматирования (тёмно-серые,
// под светлую тему): B I U S и «Aa» для выбора шрифта.
var formatIcons = map[string]string{
	"bold": `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24">
		<path d="M8 4.5h5.2a3.5 3.5 0 0 1 0 7H8z M8 11.5h6a3.75 3.75 0 0 1 0 7.5H8z"
		 fill="none" stroke="#444444" stroke-width="1.9" stroke-linejoin="round"/></svg>`,
	"italic": `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24">
		<line x1="14.5" y1="4.5" x2="10.5" y2="19.5" stroke="#444444" stroke-width="1.9" stroke-linecap="round"/>
		<line x1="6.5" y1="4.5" x2="14" y2="4.5" stroke="#444444" stroke-width="1.9" stroke-linecap="round"/>
		<line x1="10" y1="19.5" x2="17.5" y2="19.5" stroke="#444444" stroke-width="1.9" stroke-linecap="round"/></svg>`,
	"underline": `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24">
		<path d="M7.5 4v7a4.5 4.5 0 0 0 9 0V4 M6.5 20h11"
		 fill="none" stroke="#444444" stroke-width="1.9" stroke-linecap="round"/></svg>`,
	"strike": `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24">
		<path d="M8 4.5h5.2a3.5 3.5 0 0 1 0 7H8z M8 11.5h6a3.75 3.75 0 0 1 0 7.5H8z"
		 fill="none" stroke="#444444" stroke-width="1.7" stroke-linejoin="round"/>
		<line x1="4" y1="12" x2="20" y2="12" stroke="#444444" stroke-width="1.7" stroke-linecap="round"/></svg>`,
	"font": `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24">
		<path d="M4 18.5 9.5 5.5 15 18.5 M6.2 14h6.6" fill="none" stroke="#444444" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"/>
		<path d="M14.5 18.5 18 10.5l3.5 8 M15.7 16h4.6" fill="none" stroke="#444444" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"/></svg>`,
	"fontsize": `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24">
		<path d="M3.5 17 8 6.5 12.5 17 M5.3 13.5h5.4" fill="none" stroke="#444444" stroke-width="1.7" stroke-linecap="round" stroke-linejoin="round"/>
		<path d="M14 17l3-7 3 7 M15.2 14.8h3.6" fill="none" stroke="#444444" stroke-width="1.3" stroke-linecap="round" stroke-linejoin="round"/>
		<line x1="20.5" y1="5.5" x2="20.5" y2="19" stroke="#444444" stroke-width="1.3" stroke-linecap="round"/>
		<line x1="18.5" y1="5.5" x2="22.5" y2="5.5" stroke="#444444" stroke-width="1.3" stroke-linecap="round"/></svg>`,
	"numlist": `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24">
		<line x1="9" y1="6" x2="20.5" y2="6" stroke="#444444" stroke-width="1.8" stroke-linecap="round"/>
		<line x1="9" y1="12" x2="20.5" y2="12" stroke="#444444" stroke-width="1.8" stroke-linecap="round"/>
		<line x1="9" y1="18" x2="20.5" y2="18" stroke="#444444" stroke-width="1.8" stroke-linecap="round"/>
		<path d="M3.2 4.2 4.6 3.5v5" fill="none" stroke="#444444" stroke-width="1.3" stroke-linecap="round" stroke-linejoin="round"/>
		<path d="M3 9.2h2.6l-2.6 3h2.6" fill="none" stroke="#444444" stroke-width="1.3" stroke-linecap="round" stroke-linejoin="round"/>
		<path d="M3 14.5h2.6v3.2H3z" fill="none" stroke="#444444" stroke-width="1.3" stroke-linecap="round" stroke-linejoin="round"/>
		<path d="M3 17.7l2.6-1.5" fill="none" stroke="#444444" stroke-width="1.3" stroke-linecap="round"/></svg>`,
}

// formatIconResource — ресурс монохромного значка по имени.
func formatIconResource(kind string) fyne.Resource {
	svg, ok := formatIcons[kind]
	if !ok {
		return theme.ErrorIcon()
	}
	return fyne.NewStaticResource(kind+"-fmt.svg", []byte(svg))
}

// defaultIconColor — подложка, если цвет в конфиге не задан.
const defaultIconColor = "#607D8B"

// iconResource собирает SVG-значок: скруглённый квадрат цвета color
// с белой контурной пиктограммой kind. Неизвестный kind → globe.
func iconResource(kind, color string) fyne.Resource {
	pict, ok := builtinIcons[kind]
	if !ok {
		pict = builtinIcons["globe"]
		kind = "globe"
	}
	if color == "" {
		color = defaultIconColor
	}
	svg := fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24">
	<rect width="24" height="24" rx="5" fill="%s"/>
	<g fill="none" stroke="#ffffff" stroke-width="1.7" stroke-linecap="round" stroke-linejoin="round">%s</g>
</svg>`, color, pict)
	return fyne.NewStaticResource(kind+".svg", []byte(svg))
}

// isIconKind — встроенное имя значка, а не путь к файлу.
func isIconKind(s string) bool {
	_, ok := builtinIcons[s]
	return ok
}

// targetResource выбирает значок для кнопки приложения:
// файл из конфига → иконка exe → встроенный SVG.
func targetResource(t AppTarget) fyne.Resource {
	if t.Icon != "" && !isIconKind(t.Icon) && strings.ContainsAny(t.Icon, `\/.`) {
		if res, err := fyne.LoadResourceFromPath(t.Icon); err == nil {
			return res
		}
	}
	if t.Exe != "" {
		exe := t.Exe
		if !filepath.IsAbs(exe) {
			// SHGetFileInfo не ищет по PATH — раскрываем относительное имя
			exe = win.SearchPath(exe)
		}
		if png := win.ExeIconPNG(exe); png != nil {
			return fyne.NewStaticResource(t.Name+".png", png)
		}
	}
	return iconResource(t.Icon, t.Color)
}
