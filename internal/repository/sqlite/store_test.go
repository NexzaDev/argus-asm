package sqlite

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/NexzaDev/argus-asm/internal/domain"
	"github.com/NexzaDev/argus-asm/internal/repository"
)

func openTestStore(t *testing.T) *Store {
	t.Helper()
	ctx := context.Background()
	store, err := Open(ctx, filepath.Join(t.TempDir(), "argus.db"), true)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })
	if err := store.Migrate(ctx); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return store
}

func TestMigrateIsIdempotentAndReachesCurrentVersion(t *testing.T) {
	store := openTestStore(t)
	ctx := context.Background()
	if err := store.Migrate(ctx); err != nil {
		t.Fatalf("second migrate: %v", err)
	}
	version, err := store.SchemaVersion(ctx)
	if err != nil || version != 4 {
		t.Fatalf("schema version = %d, %v; want 4", version, err)
	}
}

func TestOpenMissingDatabaseReportsNotInitialized(t *testing.T) {
	_, err := Open(context.Background(), filepath.Join(t.TempDir(), "absent.db"), false)
	if !errors.Is(err, repository.ErrNotInitialized) {
		t.Fatalf("err = %v, want ErrNotInitialized", err)
	}
}

func TestDuplicateScopeMapsToDomainError(t *testing.T) {
	store := openTestStore(t)
	ctx := context.Background()
	scope := domain.Scope{Kind: domain.ScopeKindDomain, Value: "example.test", CreatedAt: time.Now().UTC()}
	if _, err := store.CreateScope(ctx, scope); err != nil {
		t.Fatalf("first insert: %v", err)
	}
	if _, err := store.CreateScope(ctx, scope); !errors.Is(err, domain.ErrDuplicate) {
		t.Fatalf("second insert: err = %v, want ErrDuplicate", err)
	}
}

func TestFindAssetNotFound(t *testing.T) {
	store := openTestStore(t)
	if _, err := store.FindAsset(context.Background(), domain.AssetKindIP, "192.0.2.1"); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
}

func TestAuditChainPersistsAcrossReopen(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "argus.db")
	store, err := Open(ctx, path, true)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	at := time.Date(2026, 10, 28, 9, 0, 0, 0, time.UTC)
	if err := store.AppendAudit(ctx, domain.AuditEntry{At: at, Actor: "local", Action: "scope.add", Detail: "domain a"}); err != nil {
		t.Fatal(err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	reopened, err := Open(ctx, path, false)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	if err := reopened.AppendAudit(ctx, domain.AuditEntry{At: at.Add(time.Second), Actor: "local", Action: "scope.add", Detail: "domain b"}); err != nil {
		t.Fatal(err)
	}
	entries, err := reopened.ListAudit(ctx)
	if err != nil {
		t.Fatal(err)
	}
	result := domain.VerifyAuditChain(entries)
	if !result.OK || result.Chained != 2 {
		t.Fatalf("verification = %+v", result)
	}
}

func TestAuditLogIsAppendOnly(t *testing.T) {
	store := openTestStore(t)
	ctx := context.Background()
	at := time.Date(2026, 10, 28, 9, 0, 0, 0, time.UTC)
	if err := store.AppendAudit(ctx, domain.AuditEntry{At: at, Actor: "local", Action: "scope.add", Detail: "domain a"}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.db.ExecContext(ctx, `UPDATE audit_log SET detail = 'forged'`); err == nil {
		t.Fatal("UPDATE on audit_log must be refused")
	}
	if _, err := store.db.ExecContext(ctx, `DELETE FROM audit_log`); err == nil {
		t.Fatal("DELETE on audit_log must be refused")
	}
	entries, err := store.ListAudit(ctx)
	if err != nil || len(entries) != 1 || entries[0].Detail != "domain a" {
		t.Fatalf("entries = %+v, %v", entries, err)
	}
}
