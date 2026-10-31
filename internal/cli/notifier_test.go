package cli

import (
	"strings"
	"testing"
)

func TestNotifierInstallRequiresExplicitConsent(t *testing.T) {
	h := newHarness(t)
	code, stdout, _ := h.run("notifier", "install")
	if code == 0 {
		t.Fatal("install without --yes must fail")
	}
	if !strings.Contains(stdout, "explicit consent") && !strings.Contains(stdout, "--yes") {
		t.Fatalf("consent text missing: %q", stdout)
	}
	if !strings.Contains(stdout, "read-only") || !strings.Contains(stdout, "no network") {
		t.Fatalf("consent text must state read-only and no network: %q", stdout)
	}
}

func TestNotifierRejectsUnsafeIntervalAndUnknownSubcommand(t *testing.T) {
	h := newHarness(t)
	if code, _, _ := h.run("notifier", "run", "--interval", "1s"); code == 0 {
		t.Fatal("interval below minimum accepted")
	}
	if code, _, _ := h.run("notifier", "run", "--detail", "everything"); code == 0 {
		t.Fatal("unknown detail accepted")
	}
	if code, _, _ := h.run("notifier", "daemonize"); code == 0 {
		t.Fatal("unknown subcommand accepted")
	}
	if code, _, _ := h.run("notifier"); code == 0 {
		t.Fatal("missing subcommand accepted")
	}
}

func TestNotifierStatusReportsWithoutWriting(t *testing.T) {
	h := newHarness(t)
	code, stdout, stderr := h.run("notifier", "status")
	if code != 0 {
		t.Fatalf("status exit %d: %s", code, stderr)
	}
	if !strings.Contains(stdout, "autostart:") || !strings.Contains(stdout, "os notifications:") {
		t.Fatalf("status output = %q", stdout)
	}
}
