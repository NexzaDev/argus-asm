package cli

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"strings"

	"github.com/NexzaDev/argus-asm/internal/domain"
	"github.com/NexzaDev/argus-asm/internal/repository/sqlite"
	"github.com/NexzaDev/argus-asm/internal/vault"
)

const maxSealedBytes = vault.MaxPlaintext + 1<<20

func (a *App) runDB(ctx context.Context, args []string) error {
	if a.inConsole {
		return usagef("db is not available in the console, because the passphrase is read from standard input; run it from the shell")
	}
	if len(args) == 0 {
		return usagef("db requires a subcommand: seal or unseal")
	}
	rest := args[1:]
	switch args[0] {
	case "seal":
		return a.runDBSeal(ctx, rest)
	case "unseal":
		return a.runDBUnseal(ctx, rest)
	default:
		return usagef("unknown db subcommand %q", args[0])
	}
}

func (a *App) runDBSeal(ctx context.Context, args []string) error {
	flags := newFlagSet("db seal", a.stdout, "db seal --out FILE --passphrase-stdin [--db PATH] [--actor NAME]")
	common := addCommon(flags)
	out := flags.String("out", "", "sealed file to create (must not exist; created with mode 0600)")
	passStdin := flags.Bool("passphrase-stdin", false, "read the passphrase from the first line of standard input")
	if _, err := parseArgs(flags, args); err != nil {
		return err
	}
	if strings.TrimSpace(*out) == "" {
		return usagef("db seal requires --out FILE")
	}
	if !*passStdin {
		return usagef("db seal requires --passphrase-stdin; the passphrase is never read from arguments or environment variables")
	}
	passphrase, err := a.readPassphrase()
	if err != nil {
		return err
	}
	file, err := createNewFile(*out)
	if err != nil {
		return err
	}
	written := false
	defer func() {
		if !written {
			_ = file.Close()
			_ = os.Remove(*out)
		}
	}()
	sess, err := a.openSession(ctx, common)
	if err != nil {
		return err
	}
	defer sess.close()
	snapshot, err := sess.svc.SnapshotDatabase(ctx, sess.actor)
	if err != nil {
		return err
	}
	sealed, err := vault.Seal(snapshot, passphrase, vault.DefaultIterations)
	if err != nil {
		return err
	}
	if _, err := file.Write(sealed); err != nil {
		return fmt.Errorf("write sealed file: %w", err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("close sealed file: %w", err)
	}
	written = true
	fmt.Fprintf(a.stdout, "sealed database to %s (%d bytes)\n", *out, len(sealed))
	return nil
}

func (a *App) runDBUnseal(ctx context.Context, args []string) error {
	flags := newFlagSet("db unseal", a.stdout, "db unseal --in FILE --out FILE --passphrase-stdin")
	in := flags.String("in", "", "sealed file to read")
	out := flags.String("out", "", "database file to create (must not exist; created with mode 0600)")
	passStdin := flags.Bool("passphrase-stdin", false, "read the passphrase from the first line of standard input")
	if _, err := parseArgs(flags, args); err != nil {
		return err
	}
	if strings.TrimSpace(*in) == "" || strings.TrimSpace(*out) == "" {
		return usagef("db unseal requires --in FILE and --out FILE")
	}
	if !*passStdin {
		return usagef("db unseal requires --passphrase-stdin")
	}
	passphrase, err := a.readPassphrase()
	if err != nil {
		return err
	}
	data, err := readSealed(*in)
	if err != nil {
		return err
	}
	plain, err := vault.Open(data, passphrase)
	if err != nil {
		return fmt.Errorf("%w: %v", domain.ErrValidation, err)
	}
	file, err := createNewFile(*out)
	if err != nil {
		return err
	}
	if _, err := file.Write(plain); err != nil {
		_ = file.Close()
		_ = os.Remove(*out)
		return fmt.Errorf("write database: %w", err)
	}
	if err := file.Close(); err != nil {
		_ = os.Remove(*out)
		return fmt.Errorf("close database: %w", err)
	}
	store, err := sqlite.Open(ctx, *out, false)
	if err == nil {
		defer store.Close()
		var version int
		version, err = store.SchemaVersion(ctx)
		if err == nil && version == 0 {
			err = errors.New("no schema migrations recorded")
		}
		if err == nil {
			fmt.Fprintf(a.stdout, "unsealed database to %s (schema version %d); run 'init' to upgrade if needed\n", *out, version)
			return nil
		}
	}
	_ = os.Remove(*out)
	return fmt.Errorf("%w: unsealed data is not an argus-asm database: %v", domain.ErrValidation, err)
}

func (a *App) readPassphrase() ([]byte, error) {
	if a.in == nil {
		return nil, fmt.Errorf("%w: no standard input for the passphrase", domain.ErrValidation)
	}
	line, err := bufio.NewReaderSize(a.in, 1024).ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return nil, fmt.Errorf("read passphrase: %w", err)
	}
	passphrase := strings.TrimRight(line, "\r\n")
	if passphrase == "" {
		return nil, fmt.Errorf("%w: the passphrase on standard input is empty", domain.ErrValidation)
	}
	return []byte(passphrase), nil
}

func createNewFile(path string) (*os.File, error) {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if errors.Is(err, fs.ErrExist) {
		return nil, fmt.Errorf("%w: %s already exists; choose a new file name", domain.ErrValidation, path)
	}
	if err != nil {
		return nil, fmt.Errorf("create %s: %w", path, err)
	}
	return file, nil
}

func readSealed(path string) ([]byte, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("inspect sealed file: %w", err)
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("%w: %s is not a regular file", domain.ErrValidation, path)
	}
	if info.Size() > maxSealedBytes {
		return nil, fmt.Errorf("%w: sealed file is larger than %d bytes", domain.ErrValidation, maxSealedBytes)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read sealed file: %w", err)
	}
	return data, nil
}
