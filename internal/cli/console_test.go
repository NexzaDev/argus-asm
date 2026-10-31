package cli

import (
	"bytes"
	"context"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestSplitLineHandlesQuotes(t *testing.T) {
	got, err := splitLine(`findings list --asset "www example" 'a b'`)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"findings", "list", "--asset", "www example", "a b"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("tokens = %q, want %q", got, want)
	}
	if _, err := splitLine(`scope add "unterminated`); err == nil {
		t.Fatal("unterminated quote must fail")
	}
}

func TestConsoleRunsCommandsAndSurvivesErrors(t *testing.T) {
	h := newHarness(t)
	h.expect(ExitOK, h.withDB("init")...)
	script := strings.Join([]string{
		"# comment lines are ignored",
		"scope add example.test",
		"asset add 203.0.113.99",
		"asset add www.example.test",
		"findings list",
		"bogus",
		"exit",
		"asset add never.example.test",
	}, "\n") + "\n"

	var stdout, stderr bytes.Buffer
	app := New(Options{
		Stdin:         strings.NewReader(script),
		Stdout:        &stdout,
		Stderr:        &stderr,
		Getenv:        func(string) string { return "" },
		UserConfigDir: func() (string, error) { return t.TempDir(), nil },
		Now:           func() time.Time { return time.Date(2026, 10, 28, 9, 0, 0, 0, time.UTC) },
	})
	code := app.Run(context.Background(), []string{"console", "--db", h.dbPath})
	if code != ExitOK {
		t.Fatalf("console exit = %d, stderr = %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "added domain asset www.example.test") {
		t.Fatalf("expected in-console asset add, stdout:\n%s", stdout.String())
	}
	if strings.Contains(stdout.String(), "never.example.test") {
		t.Fatal("commands after exit must not run")
	}
	if !strings.Contains(stderr.String(), "outside every authorized scope") {
		t.Fatalf("expected out-of-scope error to be reported, stderr:\n%s", stderr.String())
	}
	if !strings.Contains(stderr.String(), "unknown command") {
		t.Fatalf("expected unknown-command error, stderr:\n%s", stderr.String())
	}
}

func TestConsoleRejectsStdinInputAndNesting(t *testing.T) {
	h := newHarness(t)
	h.expect(ExitOK, h.withDB("init")...)
	var stdout, stderr bytes.Buffer
	app := New(Options{
		Stdin:         strings.NewReader("findings import --source s -\nconsole\nexit\n"),
		Stdout:        &stdout,
		Stderr:        &stderr,
		Getenv:        func(string) string { return "" },
		UserConfigDir: func() (string, error) { return t.TempDir(), nil },
		Now:           time.Now,
	})
	if code := app.Run(context.Background(), []string{"console", "--db", h.dbPath}); code != ExitOK {
		t.Fatalf("exit = %d", code)
	}
	if !strings.Contains(stderr.String(), "not available inside the console") {
		t.Fatalf("stdin in console not rejected, stderr:\n%s", stderr.String())
	}
	if !strings.Contains(stderr.String(), "cannot be started inside the console") {
		t.Fatalf("nested console not rejected, stderr:\n%s", stderr.String())
	}
}
