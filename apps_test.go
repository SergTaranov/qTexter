package main

import (
	"path/filepath"
	"testing"
)

func TestSubstText(t *testing.T) {
	cases := []struct{ tmpl, text, want string }{
		{"https://x.com/?q={text}", "привет мир", "https://x.com/?q=%D0%BF%D1%80%D0%B8%D0%B2%D0%B5%D1%82+%D0%BC%D0%B8%D1%80"},
		{"no placeholder", "текст", "no placeholder"},
		{"{text}{text}", "а", "%D0%B0%D0%B0"},
	}
	for _, c := range cases {
		if got := substText(c.tmpl, c.text); got != c.want {
			t.Errorf("substText(%q, %q) = %q, want %q", c.tmpl, c.text, got, c.want)
		}
	}
}

func TestHasPlaceholder(t *testing.T) {
	if !(AppTarget{URL: "https://x/?q={text}"}).HasPlaceholder() {
		t.Error("URL с {text} должен давать true")
	}
	if !(AppTarget{Exe: "x.exe", Args: []string{"{text}"}}).HasPlaceholder() {
		t.Error("аргумент с {text} должен давать true")
	}
	if (AppTarget{Exe: "x.exe"}).HasPlaceholder() {
		t.Error("без {text} должен быть false")
	}
}

func TestSaveLoadApps(t *testing.T) {
	dir := t.TempDir()
	orig := configPathOverride
	configPathOverride = filepath.Join(dir, "apps.json")
	defer func() { configPathOverride = orig }()

	apps := []AppTarget{
		{Name: "Тест", URL: "https://example.com/?q={text}", Paste: false, DelayMs: 0, Color: "#112233", Icon: "search"},
		{Name: "Блокнот", Exe: "notepad.exe", Paste: true, DelayMs: 100},
	}
	if err := SaveApps(apps); err != nil {
		t.Fatalf("SaveApps: %v", err)
	}
	loaded, err := LoadApps()
	if err != nil {
		t.Fatalf("LoadApps: %v", err)
	}
	if len(loaded) != 2 || loaded[0].Name != "Тест" || loaded[1].Exe != "notepad.exe" {
		t.Errorf("loaded = %+v", loaded)
	}
}
