package sqlite

import (
	"context"
	"io/fs"
	"path/filepath"
	"testing"

	"github.com/NexzaDev/argus-asm/migrations"
)

func TestMigrationFromSchema2KeepsObservationsAndAcceptsSARIF(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "schema2.db")
	store, err := Open(ctx, path, true)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer store.Close()

	for _, name := range []string{"0001_init.sql", "0002_lifecycle_and_access.sql"} {
		body, err := fs.ReadFile(migrations.Files, name)
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		if _, err := store.db.ExecContext(ctx, string(body)); err != nil {
			t.Fatalf("apply %s: %v", name, err)
		}
	}
	if _, err := store.db.ExecContext(ctx, `CREATE TABLE schema_migrations (version INTEGER PRIMARY KEY, name TEXT NOT NULL, applied_at TEXT NOT NULL)`); err != nil {
		t.Fatalf("schema_migrations: %v", err)
	}
	if _, err := store.db.ExecContext(ctx, `INSERT INTO schema_migrations (version, name, applied_at) VALUES (1, 'a', 'x'), (2, 'b', 'x')`); err != nil {
		t.Fatalf("record versions: %v", err)
	}
	seed := []string{
		`INSERT INTO scopes (id, kind, value, created_at) VALUES (1, 'domain', 'example.test', '2026-10-29T00:00:00Z')`,
		`INSERT INTO assets (id, scope_id, kind, value, created_at) VALUES (1, 1, 'domain', 'www.example.test', '2026-10-29T00:00:00Z')`,
		`INSERT INTO import_batches (id, created_at, format, source, accepted, created, updated) VALUES (1, '2026-10-29T00:00:00Z', 'csv', 'demo-scanner', 1, 1, 0)`,
		`INSERT INTO findings (id, asset_id, title, severity, source, dedupe_key, first_seen, last_seen) VALUES (1, 1, 'Expired TLS', 'high', 'demo-scanner', 'k1', '2026-10-29T00:00:00Z', '2026-10-29T00:00:00Z')`,
		`INSERT INTO finding_observations (batch_id, finding_id, severity) VALUES (1, 1, 'high')`,
	}
	for _, statement := range seed {
		if _, err := store.db.ExecContext(ctx, statement); err != nil {
			t.Fatalf("seed %q: %v", statement, err)
		}
	}

	if err := store.Migrate(ctx); err != nil {
		t.Fatalf("migrate to the current schema: %v", err)
	}
	version, err := store.SchemaVersion(ctx)
	if err != nil || version != 4 {
		t.Fatalf("schema version = %d, %v; want 4", version, err)
	}
	var observations int
	if err := store.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM finding_observations WHERE batch_id = 1 AND finding_id = 1`).Scan(&observations); err != nil || observations != 1 {
		t.Fatalf("observations after rebuild = %d, %v; want 1", observations, err)
	}
	var batchSource string
	if err := store.db.QueryRowContext(ctx, `SELECT source FROM import_batches WHERE id = 1`).Scan(&batchSource); err != nil || batchSource != "demo-scanner" {
		t.Fatalf("batch after rebuild = %q, %v", batchSource, err)
	}
	if _, err := store.db.ExecContext(ctx, `INSERT INTO import_batches (created_at, format, source, accepted, created, updated) VALUES ('2026-10-29T00:00:00Z', 'sarif', 'demo-sarif', 0, 0, 0)`); err != nil {
		t.Fatalf("sarif batch rejected: %v", err)
	}
	var cascaded int
	if err := store.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM finding_observations`).Scan(&cascaded); err != nil || cascaded != 1 {
		t.Fatalf("observation count = %d, %v", cascaded, err)
	}
	var fk string
	if err := store.db.QueryRowContext(ctx, `PRAGMA foreign_keys`).Scan(&fk); err != nil || fk != "1" {
		t.Fatalf("foreign_keys after migration = %q, %v; want 1", fk, err)
	}
}
