//go:build !nogui

package gui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"github.com/NexzaDev/argus-asm/internal/config"
)

func (s *session) chooser() fyne.CanvasObject {
	remember := widget.NewCheck("Remember my choice", nil)
	status := widget.NewLabel("")
	status.Wrapping = fyne.TextWrapWord
	dashboard := widget.NewButton("Dashboard (graphical window)", func() {
		if remember.Checked {
			if err := s.remember(config.UIGUI); err != nil {
				status.SetText(err.Error())
				return
			}
		}
		s.result.Choice = config.UIGUI
		s.window.SetContent(s.shell())
	})
	dashboard.Importance = widget.HighImportance
	terminal := widget.NewButton("Terminal", func() {
		if remember.Checked {
			if err := s.remember(config.UIConsole); err != nil {
				status.SetText(err.Error())
				return
			}
		}
		s.result.Choice = config.UIConsole
		s.app.Quit()
	})
	title := widget.NewLabelWithStyle("How do you want to use Argus ASM?", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	hint := widget.NewLabel("The dashboard opens a window. The terminal runs the line-based console in the terminal you started the app from. You can change this later with: argus-asm config reset-ui")
	hint.Wrapping = fyne.TextWrapWord
	return container.NewPadded(container.NewVBox(
		title,
		hint,
		dashboard,
		terminal,
		remember,
		status,
	))
}
