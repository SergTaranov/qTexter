; Inno Setup 6: инсталлятор qTexter (установка без прав администратора)
; Сборка: ISCC.exe installer\qtexter.iss  (из корня репозитория)
; Перед сборкой: build.bat в корне исходников (qTexter.exe рядом со скриптом)

#define AppName "qTexter"
#define AppVersion "1.1.0"
#define AppPublisher "SergTaranov"
#define AppExe "qTexter.exe"

[Setup]
; AppId уникален для qTexter — не менять между версиями, иначе
; обновления и удаление перестанут находить установленную копию
AppId={{937392C7-55AC-4DC5-84C8-6480728D4745}
AppName={#AppName}
AppVersion={#AppVersion}
AppPublisher={#AppPublisher}
DefaultDirName={localappdata}\Programs\{#AppName}
DisableProgramGroupPage=yes
; per-user установка: без UAC, конфиги приложения остаются рядом с exe
PrivilegesRequired=lowest
OutputDir=Output
OutputBaseFilename=qTexter-setup-v{#AppVersion}
SetupIconFile=..\icon.ico
UninstallDisplayIcon={app}\{#AppExe}
Compression=lzma2
SolidCompression=yes
WizardStyle=modern

[Languages]
Name: "russian"; MessagesFile: "compiler:Languages\Russian.isl"
Name: "english"; MessagesFile: "compiler:Default.isl"

[Tasks]
Name: "desktopicon"; Description: "{cm:CreateDesktopIcon}"; \
    GroupDescription: "{cm:AdditionalIcons}"; Flags: unchecked

[Files]
Source: "..\{#AppExe}"; DestDir: "{app}"; Flags: ignoreversion
Source: "..\README.md"; DestDir: "{app}"; Flags: ignoreversion

[Icons]
Name: "{autoprograms}\{#AppName}"; Filename: "{app}\{#AppExe}"
Name: "{autodesktop}\{#AppName}"; Filename: "{app}\{#AppExe}"; Tasks: desktopicon

[Run]
Filename: "{app}\{#AppExe}"; Description: "{cm:LaunchProgram,{#AppName}}}"; \
    Flags: nowait postinstall skipifsilent
