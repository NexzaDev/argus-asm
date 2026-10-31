package cli

import (
	"context"
	"fmt"
)

func (a *App) runInit(ctx context.Context, args []string) error {
	flags := newFlagSet("init", a.stdout, "init [--db PATH]")
	dbFlag := flags.String("db", "", "database file path")
	if _, err := parseArgs(flags, args); err != nil {
		return err
	}
	store, location, err := a.openStore(ctx, *dbFlag, true)
	if err != nil {
		return err
	}
	defer store.Close()
	if err := store.Migrate(ctx); err != nil {
		return err
	}
	version, err := store.SchemaVersion(ctx)
	if err != nil {
		return err
	}
	fmt.Fprintf(a.stdout, "database ready (schema version %d) at %s\n", version, location.Path)
	return nil
}
