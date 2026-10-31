package config

import (
	"fmt"
	"path/filepath"
	"strings"
)

const (
	EnvDBPath     = "ARGUS_ASM_DB"
	appDirName    = "argus-asm"
	dbFileName    = "argus-asm.db"
	SourceFlag    = "flag"
	SourceEnv     = "environment"
	SourceDefault = "default"
)

type Location struct {
	Path   string
	Source string
}

func ResolveDBPath(flagValue string, getenv func(string) string, userConfigDir func() (string, error)) (Location, error) {
	if value := strings.TrimSpace(flagValue); value != "" {
		return absolute(value, SourceFlag)
	}
	if value := strings.TrimSpace(getenv(EnvDBPath)); value != "" {
		return absolute(value, SourceEnv)
	}
	dir, err := userConfigDir()
	if err != nil {
		return Location{}, fmt.Errorf("determine user config directory: %w (set --db or %s)", err, EnvDBPath)
	}
	return absolute(filepath.Join(dir, appDirName, dbFileName), SourceDefault)
}

func absolute(path, source string) (Location, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return Location{}, fmt.Errorf("resolve database path: %w", err)
	}
	return Location{Path: abs, Source: source}, nil
}
