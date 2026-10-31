package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/NexzaDev/argus-asm/internal/buildinfo"
	"github.com/NexzaDev/argus-asm/internal/config"
	"github.com/NexzaDev/argus-asm/internal/repository/sqlite"
)

func (a *App) runVersion(args []string) error {
	flags := newFlagSet("version", a.stdout, "version")
	if _, err := parseArgs(flags, args); err != nil {
		return err
	}
	fmt.Fprintf(a.stdout, "%s %s\n", buildinfo.Name, buildinfo.Version)
	return nil
}

type aboutInfo struct {
	Name        string `json:"name"`
	Version     string `json:"version"`
	Description string `json:"description"`
	License     string `json:"license"`
	ProjectURL  string `json:"project_url"`
	IssuesURL   string `json:"issues_url"`
	Commit      string `json:"commit"`
	BuildDate   string `json:"build_date"`
	OS          string `json:"os"`
	Arch        string `json:"arch"`
	GoVersion   string `json:"go_version"`
}

func currentAboutInfo() aboutInfo {
	return aboutInfo{
		Name:        buildinfo.Name,
		Version:     buildinfo.Version,
		Description: buildinfo.Description,
		License:     buildinfo.License,
		ProjectURL:  buildinfo.ProjectURL,
		IssuesURL:   buildinfo.IssuesURL(),
		Commit:      buildinfo.Commit,
		BuildDate:   buildinfo.BuildDate,
		OS:          runtime.GOOS,
		Arch:        runtime.GOARCH,
		GoVersion:   runtime.Version(),
	}
}

func (a *App) runAbout(args []string) error {
	flags := newFlagSet("about", a.stdout, "about [--json]")
	jsonOut := flags.Bool("json", false, "print machine-readable JSON")
	if _, err := parseArgs(flags, args); err != nil {
		return err
	}
	info := currentAboutInfo()
	if *jsonOut {
		return writeJSON(a.stdout, info)
	}
	fmt.Fprintf(a.stdout, "%s %s\n", info.Name, info.Version)
	fmt.Fprintf(a.stdout, "  %s\n", info.Description)
	fmt.Fprintf(a.stdout, "License:       %s\n", info.License)
	fmt.Fprintf(a.stdout, "Project:       %s\n", valueOrNotConfigured(info.ProjectURL))
	fmt.Fprintf(a.stdout, "Bug reports:   %s\n", valueOrNotConfigured(info.IssuesURL))
	fmt.Fprintf(a.stdout, "Commit:        %s\n", info.Commit)
	fmt.Fprintf(a.stdout, "Built:         %s\n", info.BuildDate)
	fmt.Fprintf(a.stdout, "Platform:      %s/%s\n", info.OS, info.Arch)
	fmt.Fprintf(a.stdout, "Go runtime:    %s\n", info.GoVersion)
	return nil
}

func valueOrNotConfigured(value string) string {
	if value == "" {
		return "not configured"
	}
	return value
}

type diagnosticsReport struct {
	App      aboutInfo           `json:"app"`
	Database databaseDiagnostics `json:"database"`
}

type databaseDiagnostics struct {
	ConfigSource  string         `json:"config_source"`
	Status        string         `json:"status"`
	Path          string         `json:"path,omitempty"`
	SchemaVersion int            `json:"schema_version,omitempty"`
	Counts        map[string]int `json:"counts,omitempty"`
	Error         string         `json:"error,omitempty"`
}

func (a *App) runDiagnostics(ctx context.Context, args []string) error {
	flags := newFlagSet("diagnostics", a.stdout, "diagnostics [--db PATH] [--show-path] [--json]")
	dbFlag := flags.String("db", "", "database file path")
	showPath := flags.Bool("show-path", false, "include the database file path in the report")
	jsonOut := flags.Bool("json", false, "print machine-readable JSON")
	if _, err := parseArgs(flags, args); err != nil {
		return err
	}
	location, err := config.ResolveDBPath(*dbFlag, a.getenv, a.userConfigDir)
	if err != nil {
		return err
	}
	report := diagnosticsReport{
		App:      currentAboutInfo(),
		Database: a.inspectDatabase(ctx, location, *showPath),
	}
	if *jsonOut {
		return writeJSON(a.stdout, report)
	}
	db := report.Database
	fmt.Fprintf(a.stdout, "%s %s (%s/%s, %s)\n", report.App.Name, report.App.Version,
		report.App.OS, report.App.Arch, report.App.GoVersion)
	fmt.Fprintf(a.stdout, "database config source: %s\n", db.ConfigSource)
	fmt.Fprintf(a.stdout, "database status:        %s\n", db.Status)
	if db.Path != "" {
		fmt.Fprintf(a.stdout, "database path:          %s\n", db.Path)
	}
	if db.Status == "ok" {
		fmt.Fprintf(a.stdout, "schema version:         %d\n", db.SchemaVersion)
		fmt.Fprintf(a.stdout, "scopes / assets / findings: %d / %d / %d\n",
			db.Counts["scopes"], db.Counts["assets"], db.Counts["findings"])
	}
	if db.Error != "" {
		fmt.Fprintf(a.stdout, "database error:         %s\n", db.Error)
	}
	return nil
}

func (a *App) inspectDatabase(ctx context.Context, location config.Location, showPath bool) databaseDiagnostics {
	diag := databaseDiagnostics{ConfigSource: location.Source}
	if showPath {
		diag.Path = location.Path
	}
	fail := func(err error) databaseDiagnostics {
		diag.Status = "unreadable"
		diag.Error = redact(err.Error(), location.Path, showPath)
		return diag
	}
	if _, err := os.Stat(location.Path); os.IsNotExist(err) {
		diag.Status = "not initialized (run 'argus-asm init')"
		return diag
	} else if err != nil {
		return fail(err)
	}
	store, err := sqlite.Open(ctx, location.Path, false)
	if err != nil {
		return fail(err)
	}
	defer store.Close()
	version, err := store.SchemaVersion(ctx)
	if err != nil {
		return fail(err)
	}
	inventory, err := store.Inventory(ctx)
	if err != nil {
		return fail(err)
	}
	diag.Status = "ok"
	diag.SchemaVersion = version
	diag.Counts = map[string]int{
		"scopes":       inventory.Scopes,
		"assets":       inventory.Assets,
		"findings":     inventory.Findings,
		"services":     inventory.Services,
		"certificates": inventory.Certificates,
		"tickets":      inventory.Tickets,
		"users":        inventory.Users,
	}
	return diag
}

func redact(message, path string, showPath bool) string {
	if showPath {
		return message
	}
	return strings.ReplaceAll(message, path, filepath.Base(path)+" (path hidden; use --show-path)")
}
