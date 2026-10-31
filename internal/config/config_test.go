package config

import (
	"errors"
	"path/filepath"
	"testing"
)

func noEnv(string) string { return "" }

func TestResolveDBPathPrecedence(t *testing.T) {
	configDir := func() (string, error) { return filepath.Join("base", "config"), nil }
	env := func(key string) string {
		if key == EnvDBPath {
			return filepath.Join("env", "from.db")
		}
		return ""
	}

	flagLocation, err := ResolveDBPath(filepath.Join("flag", "x.db"), env, configDir)
	if err != nil || flagLocation.Source != SourceFlag {
		t.Fatalf("flag precedence = %+v, %v", flagLocation, err)
	}
	envLocation, err := ResolveDBPath("", env, configDir)
	if err != nil || envLocation.Source != SourceEnv {
		t.Fatalf("env precedence = %+v, %v", envLocation, err)
	}
	defaultLocation, err := ResolveDBPath("  ", noEnv, configDir)
	if err != nil || defaultLocation.Source != SourceDefault {
		t.Fatalf("default = %+v, %v", defaultLocation, err)
	}
	if filepath.Base(defaultLocation.Path) != dbFileName || filepath.Base(filepath.Dir(defaultLocation.Path)) != appDirName {
		t.Fatalf("default path = %s", defaultLocation.Path)
	}
	if !filepath.IsAbs(defaultLocation.Path) {
		t.Fatalf("default path must be absolute: %s", defaultLocation.Path)
	}
}

func TestResolveDBPathWithoutConfigDir(t *testing.T) {
	failing := func() (string, error) { return "", errors.New("no home") }
	if _, err := ResolveDBPath("", noEnv, failing); err == nil {
		t.Fatal("expected error when no config directory and no override")
	}
}
