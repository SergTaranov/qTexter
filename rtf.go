package main

import (
	"fmt"
	"strings"

	"fyne.io/fyne/v2"
)

// buildRTF собирает RTF-документ с текущими начертаниями поля.
// Кладётся в буфер обмена рядом с обычным текстом: Word и Outlook
// при вставке берут RTF и сохраняют жирный/курсив/подчёркивание.
func buildRTF(text string, style fyne.TextStyle) string {
	var b strings.Builder
	b.WriteString(`{\rtf1\ansi\deff0{\fonttbl{\f0\fnil `)
	b.WriteString(rtfEscape(appState.fontName))
	b.WriteString(`;}}\viewkind4\uc1\fs`)
	// \fs — полупункты; пробел после числа обязателен: он завершает
	// управляющее слово. Без него парсер RTF глотает ЦИФРЫ НАЧАЛА ТЕКСТА
	// как продолжение параметра («\fs2412345» — первой строки нет,
	// формат сбит)
	fmt.Fprintf(&b, "%d ", int(appState.size)*2)
	if style.Bold {
		b.WriteString(`\b`)
	}
	if style.Italic {
		b.WriteString(`\i`)
	}
	if style.Underline {
		b.WriteString(`\ul`)
	}
	if style.Strikethrough {
		b.WriteString(`\strike`)
	}

	for _, r := range text {
		switch r {
		case '\n':
			b.WriteString("\\par\n")
		case '\r':
			// пропускаем: \par уже пишет перенос
		case '\t':
			b.WriteString("\\tab ")
		case '\\', '{', '}':
			fmt.Fprintf(&b, "\\%c", r)
		default:
			if r < 128 {
				b.WriteRune(r)
			} else {
				fmt.Fprintf(&b, "\\u%d?", r)
			}
		}
	}

	if style.Bold {
		b.WriteString(`\b0`)
	}
	if style.Italic {
		b.WriteString(`\i0`)
	}
	if style.Underline {
		b.WriteString(`\ulnone`)
	}
	if style.Strikethrough {
		b.WriteString(`\strike0`)
	}
	b.WriteString("}")
	return b.String()
}

func rtfEscape(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r == '\\' || r == '{' || r == '}' {
			fmt.Fprintf(&b, "\\%c", r)
			continue
		}
		if r < 128 {
			b.WriteRune(r)
		} else {
			fmt.Fprintf(&b, "\\u%d?", r)
		}
	}
	return b.String()
}
