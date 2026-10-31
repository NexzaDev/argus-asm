package notifier

import (
	"context"
	"strings"
	"testing"
)

func TestCommandNeverUsesShellAndKeepsBodyAsArgument(t *testing.T) {
	title, body := "Argus ASM", "2 new critical findings in Argus; rm -rf $HOME && \"x\""
	for _, goos := range []string{"linux", "darwin", "windows"} {
		name, args, env, err := Command(goos, title, body)
		if err != nil {
			t.Fatalf("%s: %v", goos, err)
		}
		if name == "sh" || name == "cmd" || name == "bash" {
			t.Fatalf("%s: shell used: %s", goos, name)
		}
		joined := strings.Join(args, "\x00")
		if goos == "windows" {
			if strings.Contains(joined, body) {
				t.Fatalf("windows body must travel in the environment, not argv")
			}
			if !containsEnv(env, "ARGUS_NOTIFY_BODY="+body) {
				t.Fatalf("windows env missing body: %v", env)
			}
			continue
		}
		if !strings.Contains(joined, body) {
			t.Fatalf("%s: body not passed as a single argument", goos)
		}
	}
}

func containsEnv(env []string, want string) bool {
	for _, e := range env {
		if e == want {
			return true
		}
	}
	return false
}

func TestCommandRejectsUnknownOS(t *testing.T) {
	if _, _, _, err := Command("plan9", "t", "b"); err == nil {
		t.Fatal("expected error")
	}
}

func TestOSSenderPassesCommandToExec(t *testing.T) {
	var gotName string
	var gotArgs []string
	snd := OSSender{GOOS: "linux", Exec: func(_ context.Context, name string, args []string, _ []string) error {
		gotName, gotArgs = name, args
		return nil
	}}
	if err := snd.Send(context.Background(), "Argus ASM", "1 new high finding in Argus"); err != nil {
		t.Fatal(err)
	}
	if gotName != "notify-send" || gotArgs[len(gotArgs)-1] != "1 new high finding in Argus" {
		t.Fatalf("exec got %s %v", gotName, gotArgs)
	}
}
