package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	"github.com/NexzaDev/argus-asm/internal/config"
	"github.com/NexzaDev/argus-asm/internal/domain"
	"github.com/NexzaDev/argus-asm/internal/notifier"
	"github.com/NexzaDev/argus-asm/internal/repository/sqlite"
)

type storeSource struct {
	store *sqlite.Store
}

func (s storeSource) Notifications(ctx context.Context) ([]domain.Notification, error) {
	return s.store.ListNotifications(ctx, true)
}

func (a *App) runNotifier(ctx context.Context, args []string) error {
	if len(args) == 0 {
		return usagef("notifier requires a subcommand: run, install, uninstall, or status")
	}
	rest := args[1:]
	switch args[0] {
	case "run":
		return a.runNotifierRun(ctx, rest)
	case "install":
		return a.runNotifierInstall(ctx, rest)
	case "uninstall":
		return a.runNotifierUninstall(rest)
	case "status":
		return a.runNotifierStatus(rest)
	default:
		return usagef("unknown notifier subcommand %q", args[0])
	}
}

type notifierFlags struct {
	db       *string
	interval *time.Duration
	detail   *string
}

func addNotifierFlags(flags *flag.FlagSet) notifierFlags {
	return notifierFlags{
		db:       flags.String("db", "", "database file path (overrides "+config.EnvDBPath+")"),
		interval: flags.Duration("interval", notifier.DefaultInterval, "how often to check, between 1m and 24h"),
		detail:   flags.String("detail", string(notifier.DetailMinimal), "minimal (counts only) or detailed (first messages)"),
	}
}

func (a *App) cursorPath() (string, error) {
	dir, err := a.userConfigDir()
	if err != nil {
		return "", fmt.Errorf("resolve user config directory: %w", err)
	}
	return filepath.Join(dir, "argus-asm", "notifier-cursor.json"), nil
}

func (a *App) runNotifierRun(ctx context.Context, args []string) error {
	flags := newFlagSet("notifier run", a.stdout, "notifier run [--db PATH] [--interval 5m] [--detail minimal|detailed]")
	nf := addNotifierFlags(flags)
	if _, err := parseArgs(flags, args); err != nil {
		return err
	}
	detail, err := notifier.ParseDetail(*nf.detail)
	if err != nil {
		return usagef("%v", err)
	}
	if err := notifier.ValidateInterval(*nf.interval); err != nil {
		return usagef("%v", err)
	}
	location, err := config.ResolveDBPath(*nf.db, a.getenv, a.userConfigDir)
	if err != nil {
		return err
	}
	cursor, err := a.cursorPath()
	if err != nil {
		return err
	}
	store, err := sqlite.OpenReadOnly(ctx, location.Path)
	if err != nil {
		return err
	}
	defer store.Close()
	fmt.Fprintf(a.stdout, "notifier running: checking %s every %s (read-only)\n", location.Path, *nf.interval)
	return notifier.Run(ctx, storeSource{store: store}, notifier.NewOSSender(), cursor, detail, *nf.interval, func(format string, args ...any) {
		fmt.Fprintf(a.stderr, format+"\n", args...)
	})
}

func (a *App) runNotifierInstall(ctx context.Context, args []string) error {
	flags := newFlagSet("notifier install", a.stdout, "notifier install --yes [--db PATH] [--interval 5m] [--detail minimal|detailed]")
	nf := addNotifierFlags(flags)
	yes := flags.Bool("yes", false, "confirm the consent described above")
	if _, err := parseArgs(flags, args); err != nil {
		return err
	}
	detail, err := notifier.ParseDetail(*nf.detail)
	if err != nil {
		return usagef("%v", err)
	}
	if err := notifier.ValidateInterval(*nf.interval); err != nil {
		return usagef("%v", err)
	}
	location, err := config.ResolveDBPath(*nf.db, a.getenv, a.userConfigDir)
	if err != nil {
		return err
	}
	absDB, err := filepath.Abs(location.Path)
	if err != nil {
		return err
	}
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("locate argus-asm executable: %w", err)
	}
	home, _ := os.UserHomeDir()
	layout := notifier.Layout{
		GOOS:       runtime.GOOS,
		Home:       home,
		AppData:    a.getenv("APPDATA"),
		Executable: exe,
		Args:       []string{"notifier", "run", "--db", absDB, "--interval", nf.interval.String(), "--detail", string(detail)},
	}
	plan, err := notifier.PlanInstall(layout)
	if err != nil {
		return err
	}
	if !*yes {
		fmt.Fprintf(a.stdout, "This adds a per-user autostart entry that runs Argus ASM's notifier at login.\n")
		fmt.Fprintf(a.stdout, "  autostart file: %s\n", plan.Path)
		fmt.Fprintf(a.stdout, "  checks:         %s, read-only, every %s\n", absDB, nf.interval)
		fmt.Fprintf(a.stdout, "  shows:          %s text by default; no network calls; no admin rights\n", detail)
		fmt.Fprintln(a.stdout, "Only notifications created after installation will be announced.")
		fmt.Fprintln(a.stdout, "Re-run with --yes to install. Remove it at any time with 'argus-asm notifier uninstall'.")
		return usagef("notifier install needs explicit consent: add --yes")
	}
	store, err := sqlite.OpenReadOnly(ctx, absDB)
	if err != nil {
		return err
	}
	all, err := storeSource{store: store}.Notifications(ctx)
	_ = store.Close()
	if err != nil {
		return err
	}
	cursor, err := a.cursorPath()
	if err != nil {
		return err
	}
	if err := notifier.WriteCursor(cursor, notifier.BaselineID(all)); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(plan.Path), 0o700); err != nil {
		return fmt.Errorf("create autostart directory: %w", err)
	}
	if err := os.WriteFile(plan.Path, []byte(plan.Content), 0o600); err != nil {
		return fmt.Errorf("write autostart entry: %w", err)
	}
	fmt.Fprintf(a.stdout, "installed notifier autostart: %s\n", plan.Path)
	fmt.Fprintln(a.stdout, "it starts at your next login; start it now with: argus-asm notifier run")
	return nil
}

func (a *App) runNotifierUninstall(args []string) error {
	flags := newFlagSet("notifier uninstall", a.stdout, "notifier uninstall")
	if _, err := parseArgs(flags, args); err != nil {
		return err
	}
	path, err := a.autostartPath()
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			fmt.Fprintf(a.stdout, "notifier autostart is not installed (%s)\n", path)
			return nil
		}
		return fmt.Errorf("remove autostart entry: %w", err)
	}
	fmt.Fprintf(a.stdout, "removed notifier autostart: %s\n", path)
	return nil
}

func (a *App) runNotifierStatus(args []string) error {
	flags := newFlagSet("notifier status", a.stdout, "notifier status")
	if _, err := parseArgs(flags, args); err != nil {
		return err
	}
	path, err := a.autostartPath()
	if err != nil {
		return err
	}
	if _, err := os.Stat(path); err == nil {
		fmt.Fprintf(a.stdout, "autostart: installed (%s)\n", path)
	} else {
		fmt.Fprintf(a.stdout, "autostart: not installed (%s)\n", path)
	}
	cursor, err := a.cursorPath()
	if err != nil {
		return err
	}
	if _, err := os.Stat(cursor); err == nil {
		fmt.Fprintf(a.stdout, "last announced notification id: recorded in %s\n", cursor)
	} else {
		fmt.Fprintln(a.stdout, "last announced notification id: no baseline yet (created by install or the first run)")
	}
	name, _, _, err := notifier.Command(runtime.GOOS, notifier.Title, "")
	if err == nil {
		if _, lookErr := exec.LookPath(name); lookErr == nil {
			fmt.Fprintf(a.stdout, "os notifications: %s available\n", name)
		} else {
			fmt.Fprintf(a.stdout, "os notifications: %s not found on PATH\n", name)
		}
	} else {
		fmt.Fprintf(a.stdout, "os notifications: %v\n", err)
	}
	return nil
}

func (a *App) autostartPath() (string, error) {
	home, _ := os.UserHomeDir()
	plan, err := notifier.PlanInstall(notifier.Layout{GOOS: runtime.GOOS, Home: home, AppData: a.getenv("APPDATA"), Executable: "argus-asm", Args: []string{"notifier", "run"}})
	if err != nil {
		return "", err
	}
	return plan.Path, nil
}
