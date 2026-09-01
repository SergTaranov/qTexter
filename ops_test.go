package main

import "testing"

func TestUpperCase(t *testing.T) {
	if got := UpperCase("привет Мир"); got != "ПРИВЕТ МИР" {
		t.Errorf("UpperCase = %q", got)
	}
}

func TestLowerCase(t *testing.T) {
	if got := LowerCase("ПриВЕТ мир"); got != "привет мир" {
		t.Errorf("LowerCase = %q", got)
	}
}

func TestSentenceCase(t *testing.T) {
	cases := []struct{ in, want string }{
		{"привет мир", "Привет мир"},
		{"ПРИВЕТ МИР. ВСЕМ ПРИВЕТ!", "Привет мир. Всем привет!"},
		{"что? где… когда", "Что? Где… Когда"},
		{"", ""},
		{"уже правильно", "Уже правильно"},
	}
	for _, c := range cases {
		if got := SentenceCase(c.in); got != c.want {
			t.Errorf("SentenceCase(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestTrimLines(t *testing.T) {
	if got := TrimLines("  а б \n\n в г\t"); got != "а б\n\nв г" {
		t.Errorf("TrimLines = %q", got)
	}
}

func TestJoinLines(t *testing.T) {
	if got := JoinLines("раз\nдва\n\n  три\tчетыре "); got != "раз два три четыре" {
		t.Errorf("JoinLines = %q", got)
	}
}

func TestRemoveEmptyLines(t *testing.T) {
	if got := RemoveEmptyLines("а\n\nб\n \n\nв"); got != "а\nб\nв" {
		t.Errorf("RemoveEmptyLines = %q", got)
	}
}

func TestSortLines(t *testing.T) {
	cases := []struct{ in, want string }{
		{"бана\nяблоко\nАнанас", "Ананас\nбана\nяблоко"},
		{"ёлка\nежик\nжук", "ежик\nёлка\nжук"}, // ё на своём месте, не после я
		{"один", "один"},
		{"а\nб\n", "а\nб\n"}, // завершающий перенос сохраняется
	}
	for _, c := range cases {
		if got := SortLines(c.in); got != c.want {
			t.Errorf("SortLines(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestUniqueLines(t *testing.T) {
	if got := UniqueLines("а\nб\nа\nа\nв\nб"); got != "а\nб\nв" {
		t.Errorf("UniqueLines = %q", got)
	}
}

func TestReverse(t *testing.T) {
	if got := Reverse("абв гд"); got != "дг вба" {
		t.Errorf("Reverse = %q", got)
	}
}

func TestNumberedList(t *testing.T) {
	got := NumberedList("первая\n\nвторая\nтретья")
	want := "1. первая\n\n2. вторая\n3. третья"
	if got != want {
		t.Errorf("NumberedList = %q, want %q", got, want)
	}
	// повторное применение снимает нумерацию
	if again := NumberedList(got); again != "первая\n\nвторая\nтретья" {
		t.Errorf("NumberedList toggle = %q", again)
	}
	// пустой текст
	if NumberedList("") != "" {
		t.Error("NumberedList(\"\") != \"\"")
	}
}
