package notifier

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime"
)

type ExecFunc func(ctx context.Context, name string, args []string, env []string) error

type OSSender struct {
	GOOS string
	Exec ExecFunc
}

func NewOSSender() OSSender {
	return OSSender{GOOS: runtime.GOOS, Exec: runCommand}
}

func (s OSSender) Send(ctx context.Context, title, body string) error {
	name, args, env, err := Command(s.GOOS, title, body)
	if err != nil {
		return err
	}
	return s.Exec(ctx, name, args, env)
}

const windowsBalloonScript = `Add-Type -AssemblyName System.Windows.Forms; ` +
	`$n = New-Object System.Windows.Forms.NotifyIcon; ` +
	`$n.Icon = [System.Drawing.SystemIcons]::Information; ` +
	`$n.Visible = $true; ` +
	`$n.ShowBalloonTip(5000, $env:ARGUS_NOTIFY_TITLE, $env:ARGUS_NOTIFY_BODY, [System.Windows.Forms.ToolTipIcon]::Info); ` +
	`Start-Sleep -Seconds 6; $n.Dispose()`

func Command(goos, title, body string) (string, []string, []string, error) {
	switch goos {
	case "linux":
		return "notify-send", []string{"--app-name=" + Title, title, body}, nil, nil
	case "darwin":
		return "osascript", []string{
			"-e", "on run argv",
			"-e", "display notification (item 2 of argv) with title (item 1 of argv)",
			"-e", "end run",
			title, body,
		}, nil, nil
	case "windows":
		return "powershell", []string{"-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass", "-Command", windowsBalloonScript},
			[]string{"ARGUS_NOTIFY_TITLE=" + title, "ARGUS_NOTIFY_BODY=" + body}, nil
	default:
		return "", nil, nil, fmt.Errorf("OS notifications are not supported on %s", goos)
	}
}

func runCommand(ctx context.Context, name string, args []string, env []string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Env = append(os.Environ(), env...)
	if err := cmd.Run(); err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			return fmt.Errorf("%s is not installed or not on PATH; OS notifications are unavailable", name)
		}
		return fmt.Errorf("%s failed: %w", name, err)
	}
	return nil
}
