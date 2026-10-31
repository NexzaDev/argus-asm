package cli

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestNotifyDeliverWritesOnceAndRefusesUnsafeTargets(t *testing.T) {
	h := newHarness(t)
	h.expect(ExitOK, h.withDB("init")...)
	h.expect(ExitOK, h.withDB("user", "add", "root", "--role", "admin")...)
	h.expect(ExitOK, h.withDB("user", "add", "vera", "--role", "viewer", "--actor", "root")...)
	h.expect(ExitOK, h.withDB("scope", "add", "example.test", "--actor", "root")...)
	h.expect(ExitOK, h.withDB("asset", "add", "www.example.test", "--actor", "root")...)
	csv := writeFile(t, "findings.csv", "asset,title,severity,source\nwww.example.test,Expired TLS,high,scanner-a\n")
	h.expect(ExitOK, h.withDB("findings", "import", "--source", "scanner-a", csv, "--actor", "root")...)

	outbox := filepath.Join(t.TempDir(), "outbox.jsonl")
	out, _ := h.expect(ExitOK, h.withDB("notify", "deliver", "--dry-run", "--to", outbox, "--actor", "root")...)
	if !strings.Contains(out, "1 undelivered") {
		t.Fatalf("dry run output = %q", out)
	}
	if _, err := os.Stat(outbox); !os.IsNotExist(err) {
		t.Fatalf("dry run created the outbox: %v", err)
	}

	h.expect(ExitForbidden, h.withDB("notify", "deliver", "--to", outbox, "--actor", "vera")...)
	if _, err := os.Stat(outbox); !os.IsNotExist(err) {
		t.Fatalf("denied delivery created the outbox: %v", err)
	}

	out, _ = h.expect(ExitOK, h.withDB("notify", "deliver", "--to", outbox, "--actor", "root")...)
	if !strings.Contains(out, "delivered 1 notification") {
		t.Fatalf("deliver output = %q", out)
	}
	data, err := os.ReadFile(outbox)
	if err != nil || strings.Count(string(data), "\n") != 1 || !strings.Contains(string(data), `"kind":"finding"`) {
		t.Fatalf("outbox contents = %q, %v", data, err)
	}
	if runtime.GOOS != "windows" {
		info, _ := os.Stat(outbox)
		if info.Mode().Perm() != 0o600 {
			t.Fatalf("outbox mode = %v, want 0600", info.Mode().Perm())
		}
	}

	out, _ = h.expect(ExitOK, h.withDB("notify", "deliver", "--to", outbox, "--actor", "root")...)
	if !strings.Contains(out, "delivered 0 notification") {
		t.Fatalf("second deliver output = %q", out)
	}
	again, _ := os.ReadFile(outbox)
	if string(again) != string(data) {
		t.Fatal("second delivery appended records that were already delivered")
	}

	h.expect(ExitInvalid, h.withDB("notify", "deliver", "--to", t.TempDir(), "--actor", "root")...)

	if runtime.GOOS != "windows" {
		target := filepath.Join(t.TempDir(), "target.txt")
		if err := os.WriteFile(target, nil, 0o600); err != nil {
			t.Fatal(err)
		}
		link := filepath.Join(t.TempDir(), "link.jsonl")
		if err := os.Symlink(target, link); err == nil {
			h.expect(ExitInvalid, h.withDB("notify", "deliver", "--to", link, "--actor", "root")...)
		}
	}
}
