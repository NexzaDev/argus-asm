package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/NexzaDev/argus-asm/internal/cli"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	app := cli.New(cli.Options{
		Stdin:         os.Stdin,
		Stdout:        os.Stdout,
		Stderr:        os.Stderr,
		Getenv:        os.Getenv,
		UserConfigDir: os.UserConfigDir,
		Now:           func() time.Time { return time.Now().UTC() },
		Desktop:       newDesktop(),
	})
	code := app.Run(ctx, os.Args[1:])
	stop()
	os.Exit(code)
}
