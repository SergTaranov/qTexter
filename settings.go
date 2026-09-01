package main

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// appSettings — сохраняемые между запусками настройки: размер шрифта
// поля и размер окна. Нулевые WinW/WinH означают «размер не сохранён».
type appSettings struct {
	FontSize float32 `json:"fontSize"`
	WinW     float32 `json:"windowWidth"`
	WinH     float32 `json:"windowHeight"`
}

// settingsPathOverride позволяет тестам подменить путь к настройкам.
var settingsPathOverride string

// SettingsPath возвращает путь к settings.json: рядом с exe, а если
// каталог не доступен на запись — в %APPDATA%\qTexter.
func SettingsPath() string {
	if settingsPathOverride != "" {
		return settingsPathOverride
	}
	if exe, err := os.Executable(); err == nil {
		dir := filepath.Dir(exe)
		if isWritableDir(dir) {
			return filepath.Join(dir, "settings.json")
		}
	}
	cfg, err := os.UserConfigDir()
	if err != nil {
		return "settings.json"
	}
	return filepath.Join(cfg, "qTexter", "settings.json")
}

// LoadSettings читает settings.json; при отсутствии или повреждении
// файла возвращает значения по умолчанию.
func LoadSettings() appSettings {
	st := appSettings{FontSize: defaultFontSize}
	data, err := os.ReadFile(SettingsPath())
	if err != nil {
		return st
	}
	var loaded appSettings
	if err := json.Unmarshal(data, &loaded); err != nil {
		return st
	}
	if loaded.FontSize >= minFontSize && loaded.FontSize <= maxFontSize {
		st.FontSize = loaded.FontSize
	}
	if loaded.WinW > 0 && loaded.WinH > 0 {
		st.WinW, st.WinH = loaded.WinW, loaded.WinH
	}
	return st
}

// SaveSettings записывает settings.json с отступами — файл рассчитан
// на правку руками.
func SaveSettings(st appSettings) error {
	path := SettingsPath()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o644)
}
