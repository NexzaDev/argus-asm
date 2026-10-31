//go:build !nogui

package gui

import (
	"context"
	"errors"
	"os"
	"runtime"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"

	"github.com/NexzaDev/argus-asm/internal/config"
	"github.com/NexzaDev/argus-asm/internal/desktop"
)

const appID = "com.github.nexzadev.argus-asm"

type Runner struct{}

var _ desktop.Runner = Runner{}

func (Runner) Unavailable() string {
	if runtime.GOOS == "linux" && os.Getenv("DISPLAY") == "" && os.Getenv("WAYLAND_DISPLAY") == "" {
		return "no graphical display found (DISPLAY and WAYLAND_DISPLAY are unset)"
	}
	return ""
}

func (r Runner) Run(ctx context.Context, req desktop.Request) (desktop.Result, error) {
	if reason := r.Unavailable(); reason != "" {
		return desktop.Result{}, errors.New(reason)
	}
	a := app.NewWithID(appID)
	a.Settings().SetTheme(newArgusTheme())
	a.SetIcon(appIcon)
	w := a.NewWindow("Argus ASM")
	w.SetIcon(appIcon)
	s := newSession(ctx, a, w, req)
	if req.ShowChooser {
		w.SetContent(s.chooser())
	} else {
		s.result.Choice = config.UIGUI
		w.SetContent(s.shell())
	}
	a.Settings().AddListener(func(fyne.Settings) {
		fyne.Do(func() {
			if s.result.Choice == config.UIGUI && s.shellReady {
				w.SetContent(s.shell())
			}
		})
	})
	w.Resize(fyne.NewSize(1080, 700))
	w.ShowAndRun()
	return s.result, nil
}

func (s *session) remember(mode string) error {
	if s.req.Remember == nil {
		return nil
	}
	if err := s.req.Remember(mode); err != nil {
		return errors.New("store UI preference: " + err.Error())
	}
	return nil
}
