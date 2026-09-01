@echo off
rem Сборка shortTxt3: Fyne требует CGO, поэтому подключаем MinGW-w64 из MSYS2
setlocal
set PATH=C:\msys64\mingw64\bin;%PATH%
set CGO_ENABLED=1
go build -trimpath -ldflags "-s -w -H=windowsgui" -o shortTxt.exe .
if errorlevel 1 (
    echo Build FAILED
    exit /b 1
)
echo Build OK: shortTxt.exe
