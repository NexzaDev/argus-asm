package cli

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/NexzaDev/argus-asm/internal/config"
)

func (a *App) readStoredUI() (string, error) {
	path, err := config.UIFilePath(a.userConfigDir)
	if err != nil {
		return "", err
	}
	content, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("read UI preference: %w", err)
	}
	mode, err := config.ParseStoredUI(string(content))
	if err != nil {
		return "", usageError{message: err.Error() + "; run 'argus-asm config reset-ui'"}
	}
	return mode, nil
}

func (a *App) writeStoredUI(mode string) error {
	content, err := config.FormatStoredUI(mode)
	if err != nil {
		return usageError{message: err.Error()}
	}
	path, err := config.UIFilePath(a.userConfigDir)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("create config directory: %w", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		return fmt.Errorf("write UI preference: %w", err)
	}
	return nil
}

func (a *App) runConfig(ctx context.Context, args []string) error {
	if len(args) == 0 {
		return usagef("config requires a subcommand: show, set-ui, reset-ui")
	}
	switch args[0] {
	case "show":
		return a.runConfigShow(args[1:])
	case "set-ui":
		return a.runConfigSetUI(args[1:])
	case "reset-ui":
		return a.runConfigResetUI(args[1:])
	default:
		return usagef("unknown config subcommand %q", args[0])
	}
}

func (a *App) runConfigShow(args []string) error {
	if len(args) != 0 {
		return usagef("config show takes no arguments")
	}
	stored, err := a.readStoredUI()
	if err != nil {
		return err
	}
	location, err := config.ResolveUI("", a.getenv, stored)
	if err != nil {
		return usageError{message: err.Error()}
	}
	if location.Mode == "" {
		fmt.Fprintln(a.stdout, "ui: not set (the launcher asks on a desktop, or uses the console without a display)")
		return nil
	}
	fmt.Fprintf(a.stdout, "ui: %s (source: %s)\n", location.Mode, location.Source)
	return nil
}

func (a *App) runConfigSetUI(args []string) error {
	if len(args) != 1 {
		return usagef("usage: config set-ui gui|console|cli")
	}
	mode := strings.TrimSpace(args[0])
	if err := a.writeStoredUI(mode); err != nil {
		return err
	}
	fmt.Fprintf(a.stdout, "stored UI preference: %s\n", mode)
	return nil
}

func (a *App) runConfigResetUI(args []string) error {
	if len(args) != 0 {
		return usagef("config reset-ui takes no arguments")
	}
	removed, err := a.forgetUIChoiceReport()
	if err != nil {
		return err
	}
	if removed {
		fmt.Fprintln(a.stdout, "stored UI preference removed; the launcher asks again on a desktop")
	} else {
		fmt.Fprintln(a.stdout, "no stored UI preference; nothing to reset")
	}
	return nil
}

func (a *App) forgetUIChoice() error {
	_, err := a.forgetUIChoiceReport()
	return err
}

func (a *App) forgetUIChoiceReport() (bool, error) {
	path, err := config.UIFilePath(a.userConfigDir)
	if err != nil {
		return false, err
	}
	err = os.Remove(path)
	switch {
	case err == nil:
		return true, nil
	case errors.Is(err, fs.ErrNotExist):
		return false, nil
	default:
		return false, fmt.Errorf("remove UI preference: %w", err)
	}
}
