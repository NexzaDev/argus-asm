package sqlite

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/NexzaDev/argus-asm/internal/repository"
)

func TestOpenReadOnlyRejectsWrites(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "ro.db")
	rw, err := Open(ctx, path, true)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := rw.db.ExecContext(ctx, "CREATE TABLE probe (a INTEGER)"); err != nil {
		t.Fatal(err)
	}
	if err := rw.Close(); err != nil {
		t.Fatal(err)
	}
	ro, err := OpenReadOnly(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	defer ro.Close()
	if _, err := ro.db.ExecContext(ctx, "INSERT INTO probe (a) VALUES (1)"); err == nil {
		t.Fatal("write through read-only store succeeded")
	}
	var n int
	if err := ro.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM probe").Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Fatalf("rows = %d", n)
	}
}

func TestOpenReadOnlyMissingDatabase(t *testing.T) {
	_, err := OpenReadOnly(context.Background(), filepath.Join(t.TempDir(), "absent.db"))
	if !errors.Is(err, repository.ErrNotInitialized) {
		t.Fatalf("err = %v", err)
	}
}
