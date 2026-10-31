//go:build !nogui

package gui

import (
	_ "embed"

	"fyne.io/fyne/v2"
)

//go:embed assets/argus-asm-512.png
var appIconPNG []byte

var appIcon = fyne.NewStaticResource("argus-asm.png", appIconPNG)
