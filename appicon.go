package main

import (
	_ "embed"

	"fyne.io/fyne/v2"
)

//go:embed icon.png
var appIconPNG []byte

// appIcon — значок окна и панели задач.
var appIcon = fyne.NewStaticResource("icon.png", appIconPNG)
