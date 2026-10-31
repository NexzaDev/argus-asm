package cli

import (
	"bytes"
	"context"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/NexzaDev/argus-asm/internal/desktop"
	"github.com/NexzaDev/argus-asm/internal/probe"
)

type harness struct {
	t       *testing.T
	dbPath  string
	cfgDir  string
	env     map[string]string
	stdin   string
	desktop desktop.Runner
}

func newHarness(t *testing.T) *harness {
	t.Helper()
	dir := t.TempDir()
	return &harness{t: t, dbPath: filepath.Join(dir, "argus.db"), cfgDir: filepath.Join(dir, "config"), env: map[string]string{}}
}

func (h *harness) run(args ...string) (int, string, string) {
	h.t.Helper()
	var stdout, stderr bytes.Buffer
	app := New(Options{
		Stdin:  strings.NewReader(h.stdin),
		Stdout: &stdout,
		Stderr: &stderr,
		Getenv: func(key string) string {
			return h.env[key]
		},
		UserConfigDir: func() (string, error) {
			return h.cfgDir, nil
		},
		Now: func() time.Time { return time.Date(2026, 10, 28, 9, 0, 0, 0, time.UTC) },
		Dial: func(ctx context.Context, host string, port int, timeout time.Duration) (bool, error) {
			return port == 443, nil
		},
		Desktop: h.desktop,
	})
	code := app.Run(context.Background(), args)
	return code, stdout.String(), stderr.String()
}

func (h *harness) withDB(args ...string) []string {
	return append(args, "--db", h.dbPath)
}

func (h *harness) expect(want int, args ...string) (string, string) {
	h.t.Helper()
	code, out, errOut := h.run(args...)
	if code != want {
		h.t.Fatalf("argus-asm %s: exit %d, want %d\nstdout: %s\nstderr: %s", strings.Join(args, " "), code, want, out, errOut)
	}
	return out, errOut
}

func writeFile(t *testing.T, name, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestUsageAndHelpExitCodes(t *testing.T) {
	h := newHarness(t)
	h.stdin = "exit\n"
	h.expect(ExitOK)
	h.expect(ExitOK, "help")
	h.expect(ExitOK, "findings", "list", "-h")
	h.expect(ExitUsage, "does-not-exist")
	h.expect(ExitUsage, "--ui", "cli")
}

func TestUninitializedDatabaseExitsFour(t *testing.T) {
	h := newHarness(t)
	h.expect(ExitNotInit, h.withDB("findings", "list")...)
	out, _ := h.expect(ExitOK, h.withDB("diagnostics")...)
	if !strings.Contains(out, "not initialized") {
		t.Fatalf("diagnostics output = %q", out)
	}
}

func TestFullWorkflowWithRolesAndExitCodes(t *testing.T) {
	h := newHarness(t)
	h.expect(ExitOK, h.withDB("init")...)
	h.expect(ExitOK, h.withDB("scope", "add", "example.test")...)
	h.expect(ExitOK, h.withDB("scope", "add", "192.0.2.0/24")...)
	h.expect(ExitOutOfScope, h.withDB("asset", "add", "203.0.113.99")...)
	h.expect(ExitOK, h.withDB("asset", "add", "www.example.test")...)
	h.expect(ExitOK, h.withDB("asset", "add", "192.0.2.10")...)

	h.expect(ExitOK, h.withDB("user", "add", "root", "--role", "admin")...)
	h.expect(ExitOK, h.withDB("user", "add", "vera", "--role", "viewer", "--actor", "root")...)
	h.expect(ExitForbidden, h.withDB("user", "add", "eve", "--role", "admin", "--actor", "vera")...)
	h.expect(ExitForbidden, h.withDB("findings", "list")...)
	h.expect(ExitOK, h.withDB("findings", "list", "--actor", "vera")...)

	csv := writeFile(t, "findings.csv", "asset,title,severity,source\nwww.example.test,TLS expired,high,scanner-a\n192.0.2.10,SSH exposed,medium,scanner-a\n")
	h.expect(ExitOK, h.withDB("findings", "import", "--source", "scanner-a", "--dry-run", csv, "--actor", "root")...)
	h.expect(ExitForbidden, h.withDB("findings", "import", "--source", "scanner-a", csv, "--actor", "vera")...)
	h.expect(ExitOK, h.withDB("findings", "import", "--source", "scanner-a", csv, "--actor", "root")...)

	bad := writeFile(t, "bad.csv", "asset,title,severity,source\nwww.example.test,x,nope,s\n")
	h.expect(ExitInvalid, h.withDB("findings", "import", "--source", "s", bad, "--actor", "root")...)

	h.expect(ExitOK, h.withDB("findings", "status", "1", "--set", "accepted", "--reason", "isolated lab host", "--actor", "root")...)
	h.expect(ExitInvalid, h.withDB("findings", "status", "2", "--set", "accepted", "--actor", "root")...)

	out, _ := h.expect(ExitOK, h.withDB("findings", "list", "--explain", "--actor", "root")...)
	if !strings.Contains(out, "severity medium contributes 40") {
		t.Fatalf("explain output missing factors:\n%s", out)
	}

	out, _ = h.expect(ExitOK, h.withDB("export", "findings", "--format", "csv", "--actor", "root")...)
	if !strings.HasPrefix(out, "id,risk_score") {
		t.Fatalf("export header missing:\n%s", out)
	}

	h.expect(ExitOK, h.withDB("audit", "verify", "--actor", "root")...)
	h.expect(ExitForbidden, h.withDB("audit", "verify", "--actor", "vera")...)
}

func TestProbeIsOffByDefaultAndGated(t *testing.T) {
	h := newHarness(t)
	h.expect(ExitOK, h.withDB("init")...)
	h.expect(ExitOK, h.withDB("scope", "add", "192.0.2.0/24")...)
	h.expect(ExitOK, h.withDB("asset", "add", "192.0.2.10")...)

	h.expect(ExitForbidden, h.withDB("probe", "run", "192.0.2.10", "--ports", "22,443")...)
	h.expect(ExitForbidden, h.withDB("probe", "run", "192.0.2.10", "--ports", "443", "--execute")...)
	h.expect(ExitOK, h.withDB("asset", "probe", "allow", "192.0.2.10")...)
	out, _ := h.expect(ExitOK, h.withDB("probe", "run", "192.0.2.10", "--ports", "22,443")...)
	if !strings.Contains(out, "nothing was contacted") {
		t.Fatalf("plan output = %q", out)
	}
	out, _ = h.expect(ExitOK, h.withDB("probe", "run", "192.0.2.10", "--ports", "443,22", "--execute")...)
	if !strings.Contains(out, "open 443/tcp") {
		t.Fatalf("probe output = %q", out)
	}
	h.expect(ExitInvalid, h.withDB("probe", "run", "192.0.2.10", "--ports", "70000")...)
}

func TestLabRejectsPrivilegedPorts(t *testing.T) {
	h := newHarness(t)
	h.expect(ExitInvalid, "lab", "serve", "--ports", "80")
	h.expect(ExitUsage, "lab", "start")
}

func TestProbeDialerRealLoopback(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Skipf("loopback unavailable: %v", err)
	}
	defer listener.Close()
	port := listener.Addr().(*net.TCPAddr).Port

	h := newHarness(t)
	h.expect(ExitOK, h.withDB("init")...)
	h.expect(ExitOK, h.withDB("scope", "add", "127.0.0.0/16")...)
	h.expect(ExitOK, h.withDB("asset", "add", "127.0.0.1")...)
	h.expect(ExitOK, h.withDB("asset", "probe", "allow", "127.0.0.1")...)

	var stdout, stderr bytes.Buffer
	app := New(Options{
		Stdout: &stdout,
		Stderr: &stderr,
		Getenv: func(string) string { return "" },
		UserConfigDir: func() (string, error) {
			return t.TempDir(), nil
		},
		Now:  time.Now,
		Dial: probe.TCPConnect,
	})
	code := app.Run(context.Background(), h.withDB("probe", "run", "127.0.0.1", "--ports", strconv.Itoa(port), "--execute"))
	if code != ExitOK || !strings.Contains(stdout.String(), "open "+strconv.Itoa(port)+"/tcp") {
		t.Fatalf("exit %d, output %q, stderr %q", code, stdout.String(), stderr.String())
	}
}
