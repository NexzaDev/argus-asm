package cli

import (
	"context"
	"fmt"

	"github.com/NexzaDev/argus-asm/internal/config"
	"github.com/NexzaDev/argus-asm/internal/desktop"
	"github.com/NexzaDev/argus-asm/internal/service"
)

func (a *App) runLauncher(ctx context.Context, flagValue string) error {
	stored, err := a.readStoredUI()
	if err != nil {
		return err
	}
	location, err := config.ResolveUI(flagValue, a.getenv, stored)
	if err != nil {
		return usageError{message: err.Error()}
	}
	if location.Mode == config.UICLI {
		printUsage(a.stderr)
		return usagef("no command given; use --ui gui, --ui console, or set %s", config.EnvUI)
	}
	if location.Mode == config.UIConsole {
		return a.runConsole(ctx, nil)
	}
	reason := a.desktopUnavailableReason()
	if reason != "" {
		fmt.Fprintf(a.stderr, "argus-asm: %s; starting the terminal console instead.\n", reason)
		return a.runConsole(ctx, nil)
	}
	result, err := a.desktop.Run(ctx, desktop.Request{
		DBPath:      a.dbPathForDesktop(),
		Actor:       a.getenv(service.ActorEnvVar),
		Now:         a.now,
		ShowChooser: location.Mode == "",
		Remember:    a.rememberUIChoice,
		Forget:      a.forgetUIChoice,
	})
	if err != nil {
		return err
	}
	if result.Choice == config.UIConsole {
		return a.runConsole(ctx, nil)
	}
	return nil
}

func (a *App) desktopUnavailableReason() string {
	if a.desktop == nil {
		return "this build has no graphical interface (it was built with -tags nogui)"
	}
	return a.desktop.Unavailable()
}

func (a *App) dbPathForDesktop() string {
	location, err := config.ResolveDBPath("", a.getenv, a.userConfigDir)
	if err != nil {
		return ""
	}
	return location.Path
}

func (a *App) rememberUIChoice(mode string) error {
	return a.writeStoredUI(mode)
}
