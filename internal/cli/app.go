package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/NexzaDev/argus-asm/internal/config"
	"github.com/NexzaDev/argus-asm/internal/desktop"
	"github.com/NexzaDev/argus-asm/internal/domain"
	"github.com/NexzaDev/argus-asm/internal/probe"
	"github.com/NexzaDev/argus-asm/internal/repository"
	"github.com/NexzaDev/argus-asm/internal/repository/sqlite"
	"github.com/NexzaDev/argus-asm/internal/service"
)

const (
	ExitOK         = 0
	ExitFailure    = 1
	ExitUsage      = 2
	ExitInvalid    = 3
	ExitNotInit    = 4
	ExitOutOfScope = 5
	ExitForbidden  = 6
)

type Options struct {
	Stdin         io.Reader
	Stdout        io.Writer
	Stderr        io.Writer
	Getenv        func(string) string
	UserConfigDir func() (string, error)
	Now           func() time.Time
	Dial          probe.Dialer
	Desktop       desktop.Runner
}

type App struct {
	in            io.Reader
	stdout        io.Writer
	stderr        io.Writer
	getenv        func(string) string
	userConfigDir func() (string, error)
	now           func() time.Time
	dial          probe.Dialer
	desktop       desktop.Runner
	inConsole     bool
}

func New(options Options) *App {
	dial := options.Dial
	if dial == nil {
		dial = probe.TCPConnect
	}
	return &App{
		in:            options.Stdin,
		stdout:        options.Stdout,
		stderr:        options.Stderr,
		getenv:        options.Getenv,
		userConfigDir: options.UserConfigDir,
		now:           options.Now,
		dial:          dial,
		desktop:       options.Desktop,
	}
}

type usageError struct {
	message string
}

func (e usageError) Error() string {
	return e.message
}

func usagef(format string, args ...any) error {
	return usageError{message: fmt.Sprintf(format, args...)}
}

func (a *App) Run(ctx context.Context, args []string) int {
	if len(args) == 0 {
		return a.report(a.runLauncher(ctx, ""))
	}
	if args[0] == "--ui" || strings.HasPrefix(args[0], "--ui=") {
		return a.report(a.runUIFlag(ctx, args))
	}
	return a.report(a.dispatch(ctx, args[0], args[1:]))
}

func (a *App) runUIFlag(ctx context.Context, args []string) error {
	var value string
	rest := args[1:]
	if strings.HasPrefix(args[0], "--ui=") {
		value = strings.TrimPrefix(args[0], "--ui=")
	} else {
		if len(rest) == 0 {
			return usagef("--ui requires a value: console or cli")
		}
		value = rest[0]
		rest = rest[1:]
	}
	if len(rest) > 0 {
		return usagef("--ui selects the launcher and cannot be combined with a command; run 'argus-asm --ui %s' alone", value)
	}
	return a.runLauncher(ctx, value)
}

func (a *App) dispatch(ctx context.Context, command string, args []string) error {
	switch command {
	case "help", "-h", "--help":
		printUsage(a.stdout)
		return nil
	case "version":
		return a.runVersion(args)
	case "about":
		return a.runAbout(args)
	case "diagnostics":
		return a.runDiagnostics(ctx, args)
	case "init":
		return a.runInit(ctx, args)
	case "scope":
		return a.runScope(ctx, args)
	case "asset":
		return a.runAsset(ctx, args)
	case "service":
		return a.runService(ctx, args)
	case "cert":
		return a.runCert(ctx, args)
	case "probe":
		return a.runProbe(ctx, args)
	case "findings":
		return a.runFindings(ctx, args)
	case "ticket":
		return a.runTicket(ctx, args)
	case "search":
		return a.runSearch(ctx, args)
	case "notify":
		return a.runNotify(ctx, args)
	case "user":
		return a.runUser(ctx, args)
	case "audit":
		return a.runAudit(ctx, args)
	case "export":
		return a.runExport(ctx, args)
	case "console":
		return a.runConsole(ctx, args)
	case "config":
		return a.runConfig(ctx, args)
	case "notifier":
		return a.runNotifier(ctx, args)
	case "db":
		return a.runDB(ctx, args)
	case "lab":
		return a.runLab(ctx, args)
	default:
		return usagef("unknown command %q", command)
	}
}

func (a *App) report(err error) int {
	if err == nil || errors.Is(err, flag.ErrHelp) {
		return ExitOK
	}
	fmt.Fprintf(a.stderr, "error: %v\n", err)
	var usage usageError
	switch {
	case errors.As(err, &usage):
		fmt.Fprintln(a.stderr, "run 'argus-asm help' for usage")
		return ExitUsage
	case errors.Is(err, repository.ErrNotInitialized):
		fmt.Fprintln(a.stderr, "hint: run 'argus-asm init' to create the local database")
		return ExitNotInit
	case errors.Is(err, domain.ErrForbidden):
		return ExitForbidden
	case errors.Is(err, domain.ErrOutOfScope):
		return ExitOutOfScope
	case errors.Is(err, domain.ErrValidation), errors.Is(err, domain.ErrNotFound), errors.Is(err, domain.ErrDuplicate):
		return ExitInvalid
	default:
		return ExitFailure
	}
}

func newFlagSet(name string, output io.Writer, usage string) *flag.FlagSet {
	flags := flag.NewFlagSet("argus-asm "+name, flag.ContinueOnError)
	flags.SetOutput(output)
	flags.Usage = func() {
		fmt.Fprintf(output, "Usage: argus-asm %s\n\n", usage)
		flags.PrintDefaults()
	}
	return flags
}

func parseArgs(flags *flag.FlagSet, args []string) ([]string, error) {
	var positional []string
	for {
		if err := flags.Parse(args); err != nil {
			if errors.Is(err, flag.ErrHelp) {
				return nil, err
			}
			return nil, usageError{message: err.Error()}
		}
		rest := flags.Args()
		if len(rest) == 0 {
			return positional, nil
		}
		positional = append(positional, rest[0])
		args = rest[1:]
	}
}

type commonFlags struct {
	db    *string
	actor *string
}

func addCommon(flags *flag.FlagSet) commonFlags {
	return commonFlags{
		db:    flags.String("db", "", "database file path (overrides "+config.EnvDBPath+")"),
		actor: flags.String("actor", "", "acting user (overrides "+service.ActorEnvVar+")"),
	}
}

func (a *App) openStore(ctx context.Context, dbFlag string, create bool) (*sqlite.Store, config.Location, error) {
	location, err := config.ResolveDBPath(dbFlag, a.getenv, a.userConfigDir)
	if err != nil {
		return nil, config.Location{}, err
	}
	store, err := sqlite.Open(ctx, location.Path, create)
	if err != nil {
		return nil, location, err
	}
	return store, location, nil
}

type session struct {
	svc   *service.Service
	actor domain.Actor
	close func() error
}

func (a *App) openSession(ctx context.Context, common commonFlags) (session, error) {
	store, _, err := a.openStore(ctx, *common.db, false)
	if err != nil {
		return session{}, err
	}
	svc := service.New(store, a.now)
	requested := *common.actor
	if requested == "" {
		requested = a.getenv(service.ActorEnvVar)
	}
	actor, err := svc.ResolveActor(ctx, requested)
	if err != nil {
		_ = store.Close()
		return session{}, err
	}
	return session{svc: svc, actor: actor, close: store.Close}, nil
}
