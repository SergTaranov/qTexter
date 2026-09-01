package main

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/sys/windows/registry"

	"shorttxt/win"
)

// PlaceholderText подставляется в URL и аргументы запуска
// (заменяется на текст из поля, URL-encoded).
const PlaceholderText = "{text}"

// DefaultDelayMs — задержка перед автопастой, если в конфиге не задана.
const DefaultDelayMs = 300

// prePasteStable — сколько новое окно цели должно непрерывно удерживать
// фокус перед вставкой.
const prePasteStable = 300 * time.Millisecond

// officeWizardGrace — цели, у которых при сбойной активации Office
// после старта выскакивает «Мастер активации Microsoft Office» и съедает
// раннюю вставку; для них перед Ctrl+V мастер дожидаетсяcя и закрывается.
func officeWizardGrace(exe string) bool {
	base := strings.ToLower(filepath.Base(exe))
	return base == "winword.exe" || base == "outlook.exe"
}

// AppTarget — приложение в нижней панели.
type AppTarget struct {
	Name    string   `json:"name"`
	Exe     string   `json:"exe,omitempty"`     // путь к исполняемому файлу
	URL     string   `json:"url,omitempty"`     // или ссылка (откроется браузером)
	Args    []string `json:"args,omitempty"`    // аргументы командной строки
	Paste   bool     `json:"paste,omitempty"`   // посылать ли Ctrl+V после запуска
	DelayMs int      `json:"delayMs,omitempty"` // задержка перед Ctrl+V
	Color   string   `json:"color,omitempty"`   // цвет значка (#RRGGBB)
	Icon    string   `json:"icon,omitempty"`    // вид значка: search/mail/doc/chat/translate/globe или путь к файлу
	Browser string   `json:"browser,omitempty"` // браузер для url-целей: chrome/msedge/firefox/... или путь к exe
	PreKeys []string `json:"preKeys,omitempty"` // клавиши (tab, enter, ...) перед Ctrl+V, напр. чтобы попасть в нужное поле
	// WaitClass — класс окна цели: ждать его появления перед вставкой
	// (OpusApp у Word, Notepad у Блокнота), чтобы Ctrl+V не ушёл
	// в ещё не открывшееся приложение
	WaitClass string `json:"waitClass,omitempty"`
}

// configPathOverride позволяет тестам подменить путь к конфигу.
var configPathOverride string

// ConfigPath возвращает путь к apps.json: рядом с exe, а если каталог
// не доступен на запись — в %APPDATA%\shortTxt.
func ConfigPath() string {
	if configPathOverride != "" {
		return configPathOverride
	}
	if exe, err := os.Executable(); err == nil {
		dir := filepath.Dir(exe)
		if isWritableDir(dir) {
			return filepath.Join(dir, "apps.json")
		}
	}
	cfg, err := os.UserConfigDir()
	if err != nil {
		return "apps.json"
	}
	return filepath.Join(cfg, "shortTxt", "apps.json")
}

func isWritableDir(dir string) bool {
	f, err := os.CreateTemp(dir, ".shorttxt-*")
	if err != nil {
		return false
	}
	name := f.Name()
	f.Close()
	os.Remove(name)
	return true
}

// LoadApps читает apps.json; при первом запуске создаёт конфиг
// с автодетектом установленных приложений.
func LoadApps() ([]AppTarget, error) {
	path := ConfigPath()
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		apps := defaultApps()
		if serr := SaveApps(apps); serr != nil {
			return apps, fmt.Errorf("создан конфиг по умолчанию, но сохранить его не удалось: %w", serr)
		}
		return apps, nil
	}
	if err != nil {
		return nil, err
	}
	var apps []AppTarget
	if err := json.Unmarshal(data, &apps); err != nil {
		return nil, fmt.Errorf("не удалось разобрать %s: %w", path, err)
	}
	return apps, nil
}

// SaveApps записывает apps.json с отступами — файл рассчитан на правку руками.
func SaveApps(apps []AppTarget) error {
	path := ConfigPath()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(apps, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o644)
}

func defaultApps() []AppTarget {
	apps := []AppTarget{
		{Name: "Блокнот", Exe: "notepad.exe", Paste: true, WaitClass: "Notepad", DelayMs: 300},
		{Name: "Google", URL: "https://www.google.com/search?q={text}", Icon: "search", Color: "#4285F4"},
		{Name: "Переводчик", URL: "https://translate.google.com/?sl=auto&tl=ru&text={text}&op=translate", Icon: "translate", Color: "#4285F4"},
		{Name: "Gmail", URL: "https://mail.google.com/mail/?view=cm&fs=1&body={text}", Icon: "mail", Color: "#EA4335"},
		{Name: "Docs", URL: "https://docs.new", Paste: true, DelayMs: 4000, Icon: "doc", Color: "#4285F4"},
		{Name: "VK", URL: "https://vk.com/im", Icon: "chat", Color: "#0077FF"},
	}
	if p := appPathFromRegistry("winword.exe"); p != "" {
		// /w — непустой документ в новом экземпляре, /q — без заставки:
		// без них Word показывает стартовый экран, и вставлять некуда
		apps = append(apps, AppTarget{
			Name: "Word", Exe: p, Args: []string{"/q", "/w"},
			Paste: true, WaitClass: "OpusApp", DelayMs: 300,
		})
	}
	if p := appPathFromRegistry("outlook.exe"); p != "" {
		// фокус нового письма стоит в поле «Кому»: tab, tab, tab
		// переводят его в поле «Копия», затем в тему и в тело сообщения.
		// rctrl_renwnd32 — класс окон Outlook (и писем); без ожидания
		// окна письмо может ещё не открыться к моменту preKeys
		apps = append(apps, AppTarget{
			Name: "Outlook", Exe: p, Args: []string{"/c", "ipm.note"},
			Paste: true, DelayMs: 300, WaitClass: "rctrl_renwnd32",
			PreKeys: []string{"tab", "tab", "tab"},
		})
	}
	if p := findTelegram(); p != "" {
		// окно Telegram Desktop открывается на последнем чате,
		// фокус уже в поле сообщения — вставляем туда
		apps = append(apps, AppTarget{
			Name: "Telegram", Exe: p, Paste: true,
			WaitClass: "Telegram Desktop", DelayMs: 300,
		})
	}
	return apps
}

// appPathFromRegistry ищет exe в App Paths (HKLM, затем HKCU).
func appPathFromRegistry(exe string) string {
	const subkey = `SOFTWARE\Microsoft\Windows\CurrentVersion\App Paths\`
	for _, root := range []registry.Key{registry.LOCAL_MACHINE, registry.CURRENT_USER} {
		k, err := registry.OpenKey(root, subkey+exe, registry.QUERY_VALUE)
		if err != nil {
			continue
		}
		v, _, err := k.GetStringValue("")
		k.Close()
		if err == nil && v != "" {
			return win.LongPath(v)
		}
	}
	return ""
}

func findTelegram() string {
	candidates := []string{
		filepath.Join(`C:\Programs\Telegram`, "Telegram.exe"),
		filepath.Join(os.Getenv("LOCALAPPDATA"), "Telegram Desktop", "Telegram.exe"),
		filepath.Join(os.Getenv("ProgramFiles"), "Telegram Desktop", "Telegram.exe"),
	}
	for _, c := range candidates {
		if c == "" {
			continue
		}
		if _, err := os.Stat(c); err == nil {
			return c
		}
	}
	return ""
}

// knownBrowsers — имена браузеров, понимаемые полем browser конфига.
var knownBrowsers = []string{"chrome", "msedge", "firefox", "brave", "opera", "vivaldi", "yandex"}

// ResolveBrowser превращает имя браузера (chrome, msedge, ...) в полный путь
// к exe через App Paths реестра; путь к файлу возвращается как есть.
func ResolveBrowser(name string) string {
	if name == "" {
		return ""
	}
	if strings.ContainsAny(name, `\/`) {
		if _, err := os.Stat(name); err == nil {
			return name
		}
		return ""
	}
	if p := appPathFromRegistry(name + ".exe"); p != "" {
		return p
	}
	// запасные стандартные расположения
	locations := map[string][]string{
		"chrome": {filepath.Join(os.Getenv("ProgramFiles"), `Google\Chrome\Application\chrome.exe`),
			filepath.Join(os.Getenv("LOCALAPPDATA"), `Google\Chrome\Application\chrome.exe`)},
		"msedge": {filepath.Join(os.Getenv("ProgramFiles")+` (x86)`, `Microsoft\Edge\Application\msedge.exe`),
			filepath.Join(os.Getenv("ProgramFiles"), `Microsoft\Edge\Application\msedge.exe`)},
		"firefox": {filepath.Join(os.Getenv("ProgramFiles"), `Mozilla Firefox\firefox.exe`)},
		"yandex":  {filepath.Join(os.Getenv("LOCALAPPDATA"), `Yandex\YandexBrowser\Application\browser.exe`)},
	}
	for _, c := range locations[strings.ToLower(name)] {
		if _, err := os.Stat(c); err == nil {
			return c
		}
	}
	return ""
}

// substText заменяет {text} на текст (URL-encoded — годится и для параметров ссылки).
func substText(tmpl, text string) string {
	return strings.ReplaceAll(tmpl, PlaceholderText, url.QueryEscape(text))
}

// HasPlaceholder сообщает, подставляется ли текст прямо в команду запуска.
func (t AppTarget) HasPlaceholder() bool {
	if strings.Contains(t.URL, PlaceholderText) {
		return true
	}
	for _, a := range t.Args {
		if strings.Contains(a, PlaceholderText) {
			return true
		}
	}
	return false
}

// Launch копирует текст (и RTF-версию, если задана) в буфер обмена и
// запускает приложение; при Paste=true после ожидания окна цели и
// задержки посылает Ctrl+V активному окну.
func Launch(t AppTarget, text, rtf string) error {
	if strings.TrimSpace(text) == "" {
		return fmt.Errorf("текстовое поле пусто")
	}
	if err := win.SetClipboardTextAndRTF(text, rtf); err != nil {
		return fmt.Errorf("не удалось скопировать текст в буфер: %w", err)
	}
	var startedPID uint32 // PID запущенного нами процесса (0 — URL-цель)

	switch {
	case t.URL != "":
		target := substText(t.URL, text)
		if t.Browser != "" {
			browser := ResolveBrowser(t.Browser)
			if browser == "" {
				return fmt.Errorf("браузер %q не найден", t.Browser)
			}
			cmd := exec.Command(browser, target)
			if err := cmd.Start(); err != nil {
				return fmt.Errorf("не удалось открыть %s в %s: %w", t.Name, t.Browser, err)
			}
			go cmd.Wait()
		} else if err := win.OpenURL(target); err != nil {
			return fmt.Errorf("не удалось открыть %s: %w", t.Name, err)
		}
	case t.Exe != "":
		args := make([]string, len(t.Args))
		for i, a := range t.Args {
			args[i] = substText(a, text)
		}
		cmd := exec.Command(t.Exe, args...)
		if err := cmd.Start(); err != nil {
			return fmt.Errorf("не удалось запустить %s: %w", t.Name, err)
		}
		startedPID = uint32(cmd.Process.Pid)
		go cmd.Wait() // освобождаем описатель процесса, не дожидаясь выхода
	default:
		return fmt.Errorf("у %q не задан ни exe, ни url", t.Name)
	}

	if t.Paste {
		// ждём открытия окна цели (если задан класс); не дождались —
		// вставку отменяем, чтобы Ctrl+V не ушёл в чужое окно
		var known map[uintptr]bool
		if t.WaitClass != "" {
			known = win.WindowHandlesByClass(t.WaitClass)
			if !win.WaitForWindowClass(t.WaitClass, 20*time.Second) {
				return fmt.Errorf("окно %s (класс %q) не открылось за 20 с — вставка отменена", t.Name, t.WaitClass)
			}
		}
		delay := time.Duration(t.DelayMs) * time.Millisecond
		if delay <= 0 {
			delay = DefaultDelayMs * time.Millisecond
		}
		time.Sleep(delay)
		if t.WaitClass != "" {
			// вставлять нужно именно в НОВОЕ окно: если окна того же класса
			// уже были открыты (старый документ Word), вслепую посланный
			// Ctrl+V доставалcя в них, и текст «пропадал»
			hwnd, isNew := win.WaitForNewWindowClass(t.WaitClass, known, 5*time.Second)
			if isNew {
				// ждём непрерывного удержания фокуса новым окном (порог —
				// prePasteStable), попутно закрывая диалоги Office, если
				// они перехватывают фокус
				if !win.WaitForForegroundHandleStable(hwnd, prePasteStable, 15*time.Second, win.CloseOfficeWizards) {
					return fmt.Errorf("новое окно %s не удержало фокус — вставка отменена (мешает диалог Office?)", t.Name)
				}
			} else if !win.WaitForForegroundClass(t.WaitClass, 10*time.Second) {
				// нового окна нет (приложение переиспользовало существующее):
				// ждём активности любого окна цели
				return fmt.Errorf("окно %s не стало активным за 10 с — вставка отменена", t.Name)
			}
			if isNew && officeWizardGrace(t.Exe) && win.WindowProcessID(hwnd) == startedPID {
				// мастер активации Office выскакивает примерно через секунду
				// после старта: вставка, ушедшая РАНЬШЕ него, им съедалась.
				// Даём мастеру появиться и закрываем до вставки. Только для
				// окон нашего freshly-запущенного процесса: если окно цели
				// принадлежит уже работавшему экземпляру (Outlook), нового
				// мастера он не покажет — вставляем без ожидания
				win.CloseProcessWizard(hwnd, 1500*time.Millisecond)
				if !win.WaitForForegroundHandleStable(hwnd, prePasteStable, 5*time.Second, win.CloseOfficeWizards) {
					return fmt.Errorf("фокус не вернулся к окну %s после диалога Office — вставка отменена", t.Name)
				}
			}
			if isNew {
				// последняя проверка перед вставкой: никаких диалогов
				// Office, фокус — у нового окна цели
				win.CloseOfficeWizards()
				if win.ForegroundWindow() != hwnd {
					return fmt.Errorf("фокус у окна %s перехвачен диалогом — вставка отменена", t.Name)
				}
			}
		}
		for _, k := range t.PreKeys {
			if err := win.SendNamedKey(k); err != nil {
				return fmt.Errorf("не удалось послать %q: %w", k, err)
			}
		}
		if err := win.SendCtrlV(); err != nil {
			return fmt.Errorf("не удалось послать Ctrl+V: %w", err)
		}
	}
	return nil
}
