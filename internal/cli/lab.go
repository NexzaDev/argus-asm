package cli

import (
	"context"
	"fmt"
	"strings"

	"github.com/NexzaDev/argus-asm/internal/lab"
)

const defaultLabPorts = "18080,18022"

func (a *App) runLab(ctx context.Context, args []string) error {
	if len(args) == 0 || args[0] != "serve" {
		return usagef("lab requires the subcommand serve")
	}
	flags := newFlagSet("lab serve", a.stdout, "lab serve [--ports 18080,18022]")
	portList := flags.String("ports", defaultLabPorts, "comma-separated unprivileged ports (1024-65535, at most 8), bound to 127.0.0.1 only")
	if _, err := parseArgs(flags, args[1:]); err != nil {
		return err
	}
	ports, err := parsePortList(*portList)
	if err != nil {
		return err
	}
	if err := lab.ValidatePorts(ports); err != nil {
		return err
	}
	return lab.Serve(ctx, ports, func(addrs []string) {
		fmt.Fprintf(a.stdout, "argus-asm lab listening on %s (loopback only); press Ctrl-C to stop\n", strings.Join(addrs, ", "))
	})
}
