package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestDBSealAndUnsealRoundTrip(t *testing.T) {
	h := newHarness(t)
	h.expect(ExitOK, h.withDB("init")...)
	h.expect(ExitOK, h.withDB("user", "add", "root", "--role", "admin")...)
	h.expect(ExitOK, h.withDB("scope", "add", "example.test", "--actor", "root")...)
	h.expect(ExitOK, h.withDB("asset", "add", "www.example.test", "--actor", "root")...)

	sealed := filepath.Join(t.TempDir(), "db.sealed")
	h.stdin = "correct horse battery\n"
	h.expect(ExitOK, h.withDB("db", "seal", "--out", sealed, "--passphrase-stdin", "--actor", "root")...)
	data, err := os.ReadFile(sealed)
	if err != nil || !bytes.HasPrefix(data, []byte("ARGUSDB1")) || bytes.Contains(data, []byte("www.example.test")) {
		t.Fatalf("sealed file is not encrypted: %v", err)
	}
	if runtime.GOOS != "windows" {
		if info, _ := os.Stat(sealed); info.Mode().Perm() != 0o600 {
			t.Fatalf("sealed mode = %v, want 0600", info.Mode().Perm())
		}
	}

	restored := filepath.Join(t.TempDir(), "restored.db")
	h.stdin = "wrong passphrase\n"
	h.expect(ExitInvalid, "db", "unseal", "--in", sealed, "--out", restored, "--passphrase-stdin")
	if _, err := os.Stat(restored); !os.IsNotExist(err) {
		t.Fatalf("failed unseal left a file: %v", err)
	}

	h.stdin = "correct horse battery\n"
	out, _ := h.expect(ExitOK, "db", "unseal", "--in", sealed, "--out", restored, "--passphrase-stdin")
	if !strings.Contains(out, "schema version 4") {
		t.Fatalf("unseal output = %q", out)
	}
	h.expect(ExitInvalid, "db", "unseal", "--in", sealed, "--out", restored, "--passphrase-stdin")
}

func TestDBSealRequiresAdministratorAndStdinPassphrase(t *testing.T) {
	h := newHarness(t)
	h.expect(ExitOK, h.withDB("init")...)
	h.expect(ExitOK, h.withDB("user", "add", "root", "--role", "admin")...)
	h.expect(ExitOK, h.withDB("user", "add", "vera", "--role", "viewer", "--actor", "root")...)
	out := filepath.Join(t.TempDir(), "x.sealed")
	h.stdin = "pass\n"
	h.expect(ExitForbidden, h.withDB("db", "seal", "--out", out, "--passphrase-stdin", "--actor", "vera")...)
	h.expect(ExitUsage, h.withDB("db", "seal", "--out", out, "--actor", "root")...)
	h.stdin = "too short\n"
	h.expect(ExitInvalid, h.withDB("db", "seal", "--out", out, "--passphrase-stdin", "--actor", "root")...)
	if _, err := os.Stat(out); !os.IsNotExist(err) {
		t.Fatalf("refused seal left a file: %v", err)
	}
}

func TestDBIsRefusedInsideConsole(t *testing.T) {
	h := newHarness(t)
	h.expect(ExitOK, h.withDB("init")...)
	h.expect(ExitOK, h.withDB("user", "add", "root", "--role", "admin")...)
	out := filepath.Join(t.TempDir(), "console.sealed")
	h.stdin = "db seal --out " + out + " --passphrase-stdin\nexit\n"
	_, errOut := h.expect(ExitOK, h.withDB("console", "--actor", "root")...)
	if !strings.Contains(errOut, "not available in the console") {
		t.Fatalf("console stderr = %q", errOut)
	}
	if _, err := os.Stat(out); !os.IsNotExist(err) {
		t.Fatalf("console seal left a file: %v", err)
	}
}
