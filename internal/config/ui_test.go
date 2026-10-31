package config

import (
	"path/filepath"
	"testing"
)

func TestResolveUIPrecedence(t *testing.T) {
	env := func(key string) string {
		if key == EnvUI {
			return UIConsole
		}
		return ""
	}
	flag, err := ResolveUI(UICLI, env, UIConsole)
	if err != nil || flag != (UILocation{Mode: UICLI, Source: SourceFlag}) {
		t.Fatalf("flag precedence = %+v, %v", flag, err)
	}
	fromEnv, err := ResolveUI("", env, UIGUI)
	if err != nil || fromEnv != (UILocation{Mode: UIConsole, Source: SourceEnv}) {
		t.Fatalf("env precedence = %+v, %v", fromEnv, err)
	}
	stored, err := ResolveUI("", noEnv, UIGUI)
	if err != nil || stored != (UILocation{Mode: UIGUI, Source: SourceStored}) {
		t.Fatalf("stored = %+v, %v", stored, err)
	}
	unset, err := ResolveUI("", noEnv, "")
	if err != nil || unset != (UILocation{Source: SourceNone}) {
		t.Fatalf("unset = %+v, %v", unset, err)
	}
}

func TestResolveUIRejectsUnknownModes(t *testing.T) {
	envBad := func(key string) string {
		if key == EnvUI {
			return "tui"
		}
		return ""
	}
	if _, err := ResolveUI("tui", noEnv, ""); err == nil {
		t.Fatal("unknown flag mode must fail")
	}
	if _, err := ResolveUI("", envBad, ""); err == nil {
		t.Fatal("unknown environment mode must fail")
	}
	if _, err := ResolveUI("", noEnv, "terminal"); err == nil {
		t.Fatal("invalid stored mode must fail")
	}
}

func TestStoredUIFormatRoundTrip(t *testing.T) {
	for _, mode := range []string{UIGUI, UIConsole, UICLI} {
		content, err := FormatStoredUI(mode)
		if err != nil {
			t.Fatal(err)
		}
		parsed, err := ParseStoredUI(content)
		if err != nil || parsed != mode {
			t.Fatalf("round trip %s = %q, %v", mode, parsed, err)
		}
	}
	if empty, err := ParseStoredUI("  \n"); err != nil || empty != "" {
		t.Fatalf("blank file = %q, %v", empty, err)
	}
	if _, err := ParseStoredUI("mode=console\n"); err == nil {
		t.Fatal("malformed preference must fail")
	}
	if _, err := FormatStoredUI("tui"); err == nil {
		t.Fatal("unknown mode must not be stored")
	}
}

func TestUIFilePathUsesAppDirectory(t *testing.T) {
	path, err := UIFilePath(func() (string, error) { return filepath.Join("base", "cfg"), nil })
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(path) != uiFileName || filepath.Base(filepath.Dir(path)) != appDirName {
		t.Fatalf("ui path = %s", path)
	}
}
