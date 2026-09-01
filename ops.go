package main

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"

	"golang.org/x/text/collate"
	"golang.org/x/text/language"
)

// UpperCase переводит текст в верхний регистр.
func UpperCase(s string) string { return strings.ToUpper(s) }

// LowerCase переводит текст в нижний регистр.
func LowerCase(s string) string { return strings.ToLower(s) }

// SentenceCase приводит текст к виду предложения: всё в нижнем регистре,
// первая буква предложения (в начале текста и после . ! ? …) — заглавная.
func SentenceCase(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	capitalizeNext := true
	for _, r := range s {
		if isSentenceEnder(r) {
			capitalizeNext = true
			b.WriteRune(r)
			continue
		}
		if unicode.IsLetter(r) {
			if capitalizeNext {
				b.WriteRune(unicode.ToUpper(r))
				capitalizeNext = false
			} else {
				b.WriteRune(unicode.ToLower(r))
			}
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

func isSentenceEnder(r rune) bool {
	return r == '.' || r == '!' || r == '?' || r == '…'
}

// TrimLines убирает начальные и конечные пробелы в каждой строке.
func TrimLines(s string) string {
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimSpace(line)
	}
	return strings.Join(lines, "\n")
}

// JoinLines убирает переносы строк: склеивает текст в одну строку,
// схлопывая повторяющиеся пробелы.
func JoinLines(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

// RemoveEmptyLines удаляет пустые и состоящие из одних пробелов строки.
func RemoveEmptyLines(s string) string {
	lines := strings.Split(s, "\n")
	kept := make([]string, 0, len(lines))
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			kept = append(kept, strings.TrimRight(line, "\r"))
		}
	}
	return strings.Join(kept, "\n")
}

var ruCollator = collate.New(language.Russian)

// SortLines сортирует строки по алфавиту с учётом правил русского языка.
func SortLines(s string) string {
	trailing := strings.HasSuffix(s, "\n")
	lines := strings.Split(strings.TrimSuffix(s, "\n"), "\n")
	ruCollator.SortStrings(lines)
	res := strings.Join(lines, "\n")
	if trailing {
		res += "\n"
	}
	return res
}

// UniqueLines удаляет повторяющиеся строки, сохраняя порядок первого вхождения.
func UniqueLines(s string) string {
	trailing := strings.HasSuffix(s, "\n")
	seen := make(map[string]bool)
	lines := strings.Split(strings.TrimSuffix(s, "\n"), "\n")
	kept := make([]string, 0, len(lines))
	for _, line := range lines {
		if !seen[line] {
			seen[line] = true
			kept = append(kept, line)
		}
	}
	res := strings.Join(kept, "\n")
	if trailing {
		res += "\n"
	}
	return res
}

// Reverse переворачивает порядок символов в тексте.
func Reverse(s string) string {
	runes := []rune(s)
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}
	return string(runes)
}

var numPrefix = regexp.MustCompile(`^\s*\d+\.\s*`)

// NumberedList превращает непустые строки в нумерованный список «1. …».
// Повторное применение (большинство строк уже пронумерованы) снимает
// нумерацию — кнопка работает как переключатель.
func NumberedList(s string) string {
	lines := strings.Split(s, "\n")
	numbered, nonEmpty := 0, 0
	for _, l := range lines {
		if strings.TrimSpace(l) != "" {
			nonEmpty++
			if numPrefix.MatchString(l) {
				numbered++
			}
		}
	}
	if nonEmpty > 0 && numbered*2 >= nonEmpty {
		for i, l := range lines {
			lines[i] = numPrefix.ReplaceAllString(l, "")
		}
		return strings.Join(lines, "\n")
	}
	k := 0
	for i, l := range lines {
		if strings.TrimSpace(l) == "" {
			continue
		}
		k++
		lines[i] = fmt.Sprintf("%d. %s", k, strings.TrimLeft(l, " \t"))
	}
	return strings.Join(lines, "\n")
}
