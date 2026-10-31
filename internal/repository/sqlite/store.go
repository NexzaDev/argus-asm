package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/NexzaDev/argus-asm/internal/domain"
	"github.com/NexzaDev/argus-asm/internal/repository"
	"github.com/NexzaDev/argus-asm/migrations"
	"github.com/mattn/go-sqlite3"
)

const (
	timeLayout        = time.RFC3339Nano
	busyTimeoutMillis = 5000
	dataFileMode      = 0o600
	dataDirMode       = 0o700
)

type Store struct {
	db   *sql.DB
	path string
}

func Open(ctx context.Context, path string, create bool) (*Store, error) {
	if create {
		if err := os.MkdirAll(filepath.Dir(path), dataDirMode); err != nil {
			return nil, fmt.Errorf("create database directory: %w", err)
		}
		file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, dataFileMode)
		if err != nil {
			return nil, fmt.Errorf("create database file: %w", err)
		}
		if err := file.Close(); err != nil {
			return nil, fmt.Errorf("close database file: %w", err)
		}
	} else {
		info, err := os.Stat(path)
		if errors.Is(err, fs.ErrNotExist) {
			return nil, fmt.Errorf("%w at %s", repository.ErrNotInitialized, path)
		}
		if err != nil {
			return nil, fmt.Errorf("inspect database: %w", err)
		}
		if info.IsDir() {
			return nil, fmt.Errorf("database path %s is a directory", path)
		}
	}
	dsn := fmt.Sprintf("%s?_foreign_keys=on&_busy_timeout=%d", path, busyTimeoutMillis)
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	db.SetMaxOpenConns(1)
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("open database: %w", err)
	}
	return &Store{db: db, path: path}, nil
}

func OpenReadOnly(ctx context.Context, path string) (*Store, error) {
	info, err := os.Stat(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, fmt.Errorf("%w at %s", repository.ErrNotInitialized, path)
	}
	if err != nil {
		return nil, fmt.Errorf("inspect database: %w", err)
	}
	if info.IsDir() {
		return nil, fmt.Errorf("database path %s is a directory", path)
	}
	uri := "file:" + filepath.ToSlash(path) + "?mode=ro&_foreign_keys=on&_busy_timeout=" + fmt.Sprint(busyTimeoutMillis)
	db, err := sql.Open("sqlite3", uri)
	if err != nil {
		return nil, fmt.Errorf("open database read-only: %w", err)
	}
	db.SetMaxOpenConns(1)
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("open database read-only: %w", err)
	}
	return &Store{db: db, path: path}, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) Migrate(ctx context.Context) error {
	if _, err := s.db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (
		version    INTEGER PRIMARY KEY,
		name       TEXT NOT NULL,
		applied_at TEXT NOT NULL
	)`); err != nil {
		return fmt.Errorf("prepare schema_migrations table: %w", err)
	}
	names, err := fs.Glob(migrations.Files, "*.sql")
	if err != nil {
		return fmt.Errorf("list migrations: %w", err)
	}
	sort.Strings(names)
	applied, err := s.appliedVersions(ctx)
	if err != nil {
		return err
	}
	for _, name := range names {
		version, err := migrationVersion(name)
		if err != nil {
			return err
		}
		if applied[version] {
			continue
		}
		if err := s.applyMigration(ctx, version, name); err != nil {
			return err
		}
	}
	return nil
}

const foreignKeysOffMarker = "-- foreign_keys=off"

func checkForeignKeys(ctx context.Context, tx *sql.Tx) error {
	rows, err := tx.QueryContext(ctx, `PRAGMA foreign_key_check`)
	if err != nil {
		return fmt.Errorf("check foreign keys: %w", err)
	}
	defer rows.Close()
	if rows.Next() {
		return errors.New("foreign key violations remain after the table rebuild")
	}
	return rows.Err()
}

func migrationVersion(name string) (int, error) {
	prefix, _, ok := strings.Cut(name, "_")
	if !ok {
		return 0, fmt.Errorf("migration %s has no version prefix", name)
	}
	version, err := strconv.Atoi(prefix)
	if err != nil {
		return 0, fmt.Errorf("migration %s has invalid version prefix: %w", name, err)
	}
	return version, nil
}

func (s *Store) appliedVersions(ctx context.Context) (map[int]bool, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT version FROM schema_migrations`)
	if err != nil {
		return nil, fmt.Errorf("read applied migrations: %w", err)
	}
	defer rows.Close()
	applied := make(map[int]bool)
	for rows.Next() {
		var version int
		if err := rows.Scan(&version); err != nil {
			return nil, fmt.Errorf("scan applied migration: %w", err)
		}
		applied[version] = true
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read applied migrations: %w", err)
	}
	return applied, nil
}

func (s *Store) applyMigration(ctx context.Context, version int, name string) error {
	body, err := fs.ReadFile(migrations.Files, name)
	if err != nil {
		return fmt.Errorf("read migration %s: %w", name, err)
	}
	needsForeignKeysOff := strings.Contains(string(body), foreignKeysOffMarker)
	if needsForeignKeysOff {
		if _, err := s.db.ExecContext(ctx, `PRAGMA foreign_keys = OFF`); err != nil {
			return fmt.Errorf("disable foreign keys for migration %s: %w", name, err)
		}
		defer func() { _, _ = s.db.ExecContext(ctx, `PRAGMA foreign_keys = ON`) }()
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin migration %s: %w", name, err)
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(ctx, string(body)); err != nil {
		return fmt.Errorf("apply migration %s: %w", name, err)
	}
	if needsForeignKeysOff {
		if err := checkForeignKeys(ctx, tx); err != nil {
			return fmt.Errorf("migration %s: %w", name, err)
		}
	}
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO schema_migrations (version, name, applied_at) VALUES (?, ?, ?)`,
		version, name, formatTime(time.Now())); err != nil {
		return fmt.Errorf("record migration %s: %w", name, err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit migration %s: %w", name, err)
	}
	return nil
}

func (s *Store) SchemaVersion(ctx context.Context) (int, error) {
	var version int
	if err := s.db.QueryRowContext(ctx, `SELECT COALESCE(MAX(version), 0) FROM schema_migrations`).Scan(&version); err != nil {
		return 0, fmt.Errorf("read schema version: %w", err)
	}
	return version, nil
}

func (s *Store) Inventory(ctx context.Context) (domain.Inventory, error) {
	var inv domain.Inventory
	targets := []struct {
		table  string
		target *int
	}{
		{"scopes", &inv.Scopes},
		{"assets", &inv.Assets},
		{"findings", &inv.Findings},
		{"services", &inv.Services},
		{"certificates", &inv.Certificates},
		{"tickets", &inv.Tickets},
		{"users", &inv.Users},
	}
	for _, t := range targets {
		if err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM "+t.table).Scan(t.target); err != nil {
			return domain.Inventory{}, fmt.Errorf("count %s: %w", t.table, err)
		}
	}
	return inv, nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func formatTime(t time.Time) string {
	return t.UTC().Format(timeLayout)
}

func formatOptionalTime(t *time.Time) string {
	if t == nil {
		return ""
	}
	return formatTime(*t)
}

func parseTime(raw string) (time.Time, error) {
	t, err := time.Parse(timeLayout, raw)
	if err != nil {
		return time.Time{}, fmt.Errorf("corrupt timestamp %q: %w", raw, err)
	}
	return t, nil
}

func parseOptionalTime(raw string) (*time.Time, error) {
	if raw == "" {
		return nil, nil
	}
	t, err := parseTime(raw)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func isUniqueViolation(err error) bool {
	var sqliteErr sqlite3.Error
	return errors.As(err, &sqliteErr) && sqliteErr.ExtendedCode == sqlite3.ErrConstraintUnique
}
