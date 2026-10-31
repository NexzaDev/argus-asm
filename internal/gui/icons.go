//go:build !nogui

package gui

import (
	"bytes"
	"embed"
	"fmt"
	"image/color"

	"fyne.io/fyne/v2"
)

//go:embed icons/*.svg
var iconFiles embed.FS

func hexColor(c color.Color) string {
	r, g, b, _ := c.RGBA()
	return fmt.Sprintf("#%02X%02X%02X", r>>8, g>>8, b>>8)
}

func iconResource(name string, c color.Color) (fyne.Resource, error) {
	raw, err := iconFiles.ReadFile("icons/" + name + ".svg")
	if err != nil {
		return nil, fmt.Errorf("icon %s: %w", name, err)
	}
	colored := bytes.ReplaceAll(raw, []byte("currentColor"), []byte(hexColor(c)))
	return fyne.NewStaticResource(name+".svg", colored), nil
}

func mustIcon(name string, c color.Color) fyne.Resource {
	res, err := iconResource(name, c)
	if err != nil {
		panic(err)
	}
	return res
}
