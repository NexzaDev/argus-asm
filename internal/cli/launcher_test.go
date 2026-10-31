package cli

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/NexzaDev/argus-asm/internal/desktop"
)

type fakeDesktop struct {
	reason   string
	choice   string
	err      error
	request  desktop.Request
	runCalls int
}

func (f *fakeDesktop) Unavailable() string { return f.reason }

func (f *fakeDesktop) Run(ctx context.Context, req desktop.Request) (desktop.Result, error) {
	f.runCalls++
	f.request = req
	if req.ShowChooser && req.Remember != nil && f.choice == "console" {
		if err := req.Remember("console"); err != nil {
			return desktop.Result{}, err
		}
	}
	return desktop.Result{Choice: f.choice}, f.err
}

func TestLauncherFlagValidation(t *testing.T) {
	h := newHarness(t)
	h.expect(ExitUsage, "--ui")
	h.expect(ExitUsage, "--ui", "tui")
	h.expect(ExitUsage, "--ui=tui")
	h.expect(ExitUsage, "--ui", "console", "init", "--db", h.dbPath)
}

func TestLauncherCLIModeWithoutCommandExitsUsage(t *testing.T) {
	h := newHarness(t)
	code, _, stderr := h.run("--ui", "cli")
	if code != ExitUsage || !strings.Contains(stderr, "Usage:") {
		t.Fatalf("cli mode: exit %d stderr %q", code, stderr)
	}
}

func TestLauncherConsoleModeStartsConsole(t *testing.T) {
	h := newHarness(t)
	h.stdin = "exit\n"
	h.expect(ExitOK, "--ui", "console")
	h.stdin = "exit\n"
	h.env["ARGUS_UI"] = "console"
	h.expect(ExitOK)
}

func TestLauncherFallsBackToConsoleWithoutDisplay(t *testing.T) {
	h := newHarness(t)
	h.desktop = &fakeDesktop{reason: "no graphical display found"}
	h.stdin = "exit\n"
	code, _, stderr := h.run()
	if code != ExitOK {
		t.Fatalf("exit %d, stderr %q", code, stderr)
	}
	if !strings.Contains(stderr, "no graphical display found") || !strings.Contains(stderr, "starting the terminal console") {
		t.Fatalf("fallback reason missing: %q", stderr)
	}
}

func TestLauncherReasonsWhenBuiltWithoutGUI(t *testing.T) {
	h := newHarness(t)
	h.stdin = "exit\n"
	code, _, stderr := h.run("--ui", "gui")
	if code != ExitOK || !strings.Contains(stderr, "nogui") {
		t.Fatalf("nogui fallback: exit %d stderr %q", code, stderr)
	}
}

func TestLauncherChooserRunsDesktopAndRemembersConsole(t *testing.T) {
	h := newHarness(t)
	fake := &fakeDesktop{choice: "console"}
	h.desktop = fake
	h.stdin = "exit\n"
	h.expect(ExitOK)
	if fake.runCalls != 1 || !fake.request.ShowChooser {
		t.Fatalf("chooser not requested: %+v", fake.request)
	}
	out, _ := h.expect(ExitOK, "config", "show")
	if !strings.Contains(out, "ui: console (source: stored)") {
		t.Fatalf("remembered choice missing: %q", out)
	}
}

func TestStoredGUIPreferenceSkipsChooser(t *testing.T) {
	h := newHarness(t)
	fake := &fakeDesktop{choice: "gui"}
	h.desktop = fake
	h.expect(ExitOK, "config", "set-ui", "gui")
	h.expect(ExitOK)
	if fake.runCalls != 1 || fake.request.ShowChooser {
		t.Fatalf("stored gui must open the dashboard without the chooser: %+v", fake.request)
	}
}

func TestDesktopErrorIsReported(t *testing.T) {
	h := newHarness(t)
	h.desktop = &fakeDesktop{err: errors.New("window failed")}
	code, _, stderr := h.run("--ui", "gui")
	if code != ExitFailure || !strings.Contains(stderr, "window failed") {
		t.Fatalf("desktop error: exit %d stderr %q", code, stderr)
	}
}

func TestConfigPreferenceLifecycle(t *testing.T) {
	h := newHarness(t)
	out, _ := h.expect(ExitOK, "config", "show")
	if !strings.Contains(out, "ui: not set") {
		t.Fatalf("default show = %q", out)
	}
	h.expect(ExitOK, "config", "set-ui", "console")
	out, _ = h.expect(ExitOK, "config", "show")
	if !strings.Contains(out, "ui: console (source: stored)") {
		t.Fatalf("stored show = %q", out)
	}
	h.env["ARGUS_UI"] = "cli"
	out, _ = h.expect(ExitOK, "config", "show")
	if !strings.Contains(out, "ui: cli (source: environment)") {
		t.Fatalf("env overrides stored = %q", out)
	}
	delete(h.env, "ARGUS_UI")
	out, _ = h.expect(ExitOK, "config", "reset-ui")
	if !strings.Contains(out, "removed") {
		t.Fatalf("reset output = %q", out)
	}
	out, _ = h.expect(ExitOK, "config", "reset-ui")
	if !strings.Contains(out, "nothing to reset") {
		t.Fatalf("second reset output = %q", out)
	}
}

func TestConfigRejectsBadInput(t *testing.T) {
	h := newHarness(t)
	h.expect(ExitUsage, "config")
	h.expect(ExitUsage, "config", "unknown")
	h.expect(ExitUsage, "config", "set-ui", "tui")
	h.expect(ExitUsage, "config", "set-ui")
	h.expect(ExitUsage, "config", "show", "extra")
	h.expect(ExitUsage, "config", "reset-ui", "extra")
}

func TestMalformedStoredPreferenceIsReported(t *testing.T) {
	h := newHarness(t)
	dir := filepath.Join(h.cfgDir, "argus-asm")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "ui.conf"), []byte("ui=tui\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	code, _, stderr := h.run("config", "show")
	if code != ExitUsage || !strings.Contains(stderr, "reset-ui") {
		t.Fatalf("malformed preference: exit %d stderr %q", code, stderr)
	}
	h.expect(ExitOK, "config", "reset-ui")
}
