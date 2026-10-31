//go:build !nogui

package gui

import (
	"image/color"
	"sync/atomic"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"

	"github.com/NexzaDev/argus-asm/internal/domain"
)

type densityMode int32

const (
	densityComfortable densityMode = iota
	densityCompact
)

var currentDensity atomic.Int32

func setDensity(mode densityMode) {
	currentDensity.Store(int32(mode))
}

func getDensity() densityMode {
	return densityMode(currentDensity.Load())
}

type argusTheme struct {
	base fyne.Theme
}

func newArgusTheme() fyne.Theme {
	return &argusTheme{base: theme.DefaultTheme()}
}

func (t *argusTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	return t.base.Color(name, variant)
}

func (t *argusTheme) Font(style fyne.TextStyle) fyne.Resource {
	return t.base.Font(style)
}

func (t *argusTheme) Icon(name fyne.ThemeIconName) fyne.Resource {
	return t.base.Icon(name)
}

func (t *argusTheme) Size(name fyne.ThemeSizeName) float32 {
	base := t.base.Size(name)
	if getDensity() != densityCompact {
		return base
	}
	switch name {
	case theme.SizeNamePadding:
		return base / 2
	case theme.SizeNameInnerPadding:
		return base / 2
	}
	return base
}

func severityTextColor(severity domain.Severity, variant fyne.ThemeVariant) color.Color {
	dark := variant == theme.VariantDark
	switch severity {
	case domain.SeverityCritical:
		return pick(dark, 0xB4, 0x23, 0x18, 0xFF, 0x8F, 0x85)
	case domain.SeverityHigh:
		return pick(dark, 0x9A, 0x45, 0x00, 0xFF, 0xB7, 0x70)
	case domain.SeverityMedium:
		return pick(dark, 0x7A, 0x56, 0x00, 0xF5, 0xC4, 0x51)
	case domain.SeverityLow:
		return pick(dark, 0x1F, 0x6F, 0x43, 0x6C, 0xD4, 0x9A)
	}
	return pick(dark, 0x1D, 0x5F, 0xA8, 0x9C, 0xC4, 0xFF)
}

func pick(dark bool, lr, lg, lb, dr, dg, db uint8) color.Color {
	if dark {
		return color.NRGBA{R: dr, G: dg, B: db, A: 0xFF}
	}
	return color.NRGBA{R: lr, G: lg, B: lb, A: 0xFF}
}
