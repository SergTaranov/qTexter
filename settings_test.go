package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSaveLoadSettings(t *testing.T) {
	dir := t.TempDir()
	orig := settingsPathOverride
	settingsPathOverride = filepath.Join(dir, "settings.json")
	defer func() { settingsPathOverride = orig }()

	st := appSettings{FontSize: 16, WinW: 500, WinH: 400}
	if err := SaveSettings(st); err != nil {
		t.Fatalf("SaveSettings: %v", err)
	}
	loaded := LoadSettings()
	if loaded != st {
		t.Errorf("loaded = %+v, want %+v", loaded, st)
	}
}

func TestLoadSettingsDefaults(t *testing.T) {
	dir := t.TempDir()
	orig := settingsPathOverride
	settingsPathOverride = filepath.Join(dir, "settings.json")
	defer func() { settingsPathOverride = orig }()

	// файла нет — дефолты
	loaded := LoadSettings()
	if loaded.FontSize != defaultFontSize || loaded.WinW != 0 || loaded.WinH != 0 {
		t.Errorf("без файла loaded = %+v", loaded)
	}

	// битый JSON — дефолты
	if err := os.WriteFile(settingsPathOverride, []byte("{oops"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	loaded = LoadSettings()
	if loaded.FontSize != defaultFontSize || loaded.WinW != 0 || loaded.WinH != 0 {
		t.Errorf("битый JSON: loaded = %+v", loaded)
	}

	// значения вне диапазона игнорируются
	if err := os.WriteFile(settingsPathOverride,
		[]byte(`{"fontSize": 99, "windowWidth": -5, "windowHeight": 0}`), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	loaded = LoadSettings()
	if loaded.FontSize != defaultFontSize || loaded.WinW != 0 || loaded.WinH != 0 {
		t.Errorf("вне диапазона: loaded = %+v", loaded)
	}
}
