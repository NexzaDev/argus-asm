package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/NexzaDev/argus-asm/internal/domain"
)

func (s *Store) CreateNotifications(ctx context.Context, notifications []domain.Notification) (int, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("begin notifications: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	inserted := 0
	for _, n := range notifications {
		res, err := tx.ExecContext(ctx,
			`INSERT OR IGNORE INTO notifications (created_at, kind, message, dedupe_key) VALUES (?, ?, ?, ?)`,
			formatTime(n.CreatedAt), n.Kind, n.Message, n.DedupeKey)
		if err != nil {
			return 0, fmt.Errorf("insert notification: %w", err)
		}
		if affected, _ := res.RowsAffected(); affected == 1 {
			inserted++
		}
	}
	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("commit notifications: %w", err)
	}
	return inserted, nil
}

func insertNotification(ctx context.Context, tx *sql.Tx, n domain.Notification) error {
	if _, err := tx.ExecContext(ctx,
		`INSERT OR IGNORE INTO notifications (created_at, kind, message, dedupe_key) VALUES (?, ?, ?, ?)`,
		formatTime(n.CreatedAt), n.Kind, n.Message, n.DedupeKey); err != nil {
		return fmt.Errorf("insert notification: %w", err)
	}
	return nil
}

func (s *Store) ListNotifications(ctx context.Context, includeAcknowledged bool) ([]domain.Notification, error) {
	query := `SELECT id, created_at, kind, message, dedupe_key, acknowledged_at FROM notifications`
	if !includeAcknowledged {
		query += ` WHERE acknowledged_at = ''`
	}
	query += ` ORDER BY id`
	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list notifications: %w", err)
	}
	defer rows.Close()
	notes := []domain.Notification{}
	for rows.Next() {
		var n domain.Notification
		var createdAt, acked string
		if err := rows.Scan(&n.ID, &createdAt, &n.Kind, &n.Message, &n.DedupeKey, &acked); err != nil {
			return nil, fmt.Errorf("scan notification: %w", err)
		}
		if n.CreatedAt, err = parseTime(createdAt); err != nil {
			return nil, err
		}
		if n.AcknowledgedAt, err = parseOptionalTime(acked); err != nil {
			return nil, err
		}
		notes = append(notes, n)
	}
	return notes, rows.Err()
}

func (s *Store) AcknowledgeNotification(ctx context.Context, id int64, at time.Time) error {
	res, err := s.db.ExecContext(ctx,
		`UPDATE notifications SET acknowledged_at = ? WHERE id = ? AND acknowledged_at = ''`, formatTime(at), id)
	if err != nil {
		return fmt.Errorf("acknowledge notification: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return fmt.Errorf("%w: unacknowledged notification %d", domain.ErrNotFound, id)
	}
	return nil
}

func (s *Store) CreateUser(ctx context.Context, u domain.User) (domain.User, error) {
	res, err := s.db.ExecContext(ctx,
		`INSERT INTO users (name, role, created_at) VALUES (?, ?, ?)`, u.Name, string(u.Role), formatTime(u.CreatedAt))
	if err != nil {
		if isUniqueViolation(err) {
			return domain.User{}, fmt.Errorf("%w: user %s", domain.ErrDuplicate, u.Name)
		}
		return domain.User{}, fmt.Errorf("insert user: %w", err)
	}
	if u.ID, err = res.LastInsertId(); err != nil {
		return domain.User{}, fmt.Errorf("read user id: %w", err)
	}
	return u, nil
}

func scanUser(row rowScanner) (domain.User, error) {
	var u domain.User
	var role, createdAt string
	if err := row.Scan(&u.ID, &u.Name, &role, &createdAt); err != nil {
		return domain.User{}, err
	}
	u.Role = domain.Role(role)
	var err error
	if u.CreatedAt, err = parseTime(createdAt); err != nil {
		return domain.User{}, err
	}
	return u, nil
}

func (s *Store) FindUser(ctx context.Context, name string) (domain.User, error) {
	u, err := scanUser(s.db.QueryRowContext(ctx,
		`SELECT id, name, role, created_at FROM users WHERE name = ?`, name))
	if errors.Is(err, sql.ErrNoRows) {
		return domain.User{}, fmt.Errorf("%w: user %s", domain.ErrNotFound, name)
	}
	return u, err
}

func (s *Store) ListUsers(ctx context.Context) ([]domain.User, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, name, role, created_at FROM users ORDER BY name`)
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}
	defer rows.Close()
	users := []domain.User{}
	for rows.Next() {
		u, err := scanUser(rows)
		if err != nil {
			return nil, fmt.Errorf("scan user: %w", err)
		}
		users = append(users, u)
	}
	return users, rows.Err()
}

func (s *Store) SetUserRole(ctx context.Context, name string, role domain.Role) error {
	res, err := s.db.ExecContext(ctx, `UPDATE users SET role = ? WHERE name = ?`, string(role), name)
	if err != nil {
		return fmt.Errorf("update user role: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return fmt.Errorf("%w: user %s", domain.ErrNotFound, name)
	}
	return nil
}

func (s *Store) CountAdmins(ctx context.Context) (int, error) {
	var n int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM users WHERE role = 'admin'`).Scan(&n); err != nil {
		return 0, fmt.Errorf("count admins: %w", err)
	}
	return n, nil
}

func (s *Store) AppendAudit(ctx context.Context, entry domain.AuditEntry) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin audit append: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	var prev string
	err = tx.QueryRowContext(ctx, `SELECT hash FROM audit_log ORDER BY id DESC LIMIT 1`).Scan(&prev)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("read audit chain head: %w", err)
	}
	hash := domain.AuditHash(prev, entry.At, entry.Actor, entry.Action, entry.Detail)
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO audit_log (at, action, detail, actor, prev_hash, hash) VALUES (?, ?, ?, ?, ?, ?)`,
		formatTime(entry.At), entry.Action, entry.Detail, entry.Actor, prev, hash); err != nil {
		return fmt.Errorf("append audit entry: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit audit entry: %w", err)
	}
	return nil
}

func (s *Store) ListAudit(ctx context.Context) ([]domain.AuditEntry, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, at, actor, action, detail, prev_hash, hash FROM audit_log ORDER BY id`)
	if err != nil {
		return nil, fmt.Errorf("list audit log: %w", err)
	}
	defer rows.Close()
	entries := []domain.AuditEntry{}
	for rows.Next() {
		var e domain.AuditEntry
		var at string
		if err := rows.Scan(&e.ID, &at, &e.Actor, &e.Action, &e.Detail, &e.PrevHash, &e.Hash); err != nil {
			return nil, fmt.Errorf("scan audit entry: %w", err)
		}
		if e.At, err = parseTime(at); err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

func (s *Store) UndeliveredNotifications(ctx context.Context) ([]domain.Notification, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, created_at, kind, message, dedupe_key, acknowledged_at FROM notifications WHERE delivered_at = '' ORDER BY id`)
	if err != nil {
		return nil, fmt.Errorf("list undelivered notifications: %w", err)
	}
	defer rows.Close()
	notes := []domain.Notification{}
	for rows.Next() {
		var n domain.Notification
		var createdAt, acked string
		if err := rows.Scan(&n.ID, &createdAt, &n.Kind, &n.Message, &n.DedupeKey, &acked); err != nil {
			return nil, fmt.Errorf("scan notification: %w", err)
		}
		if n.CreatedAt, err = parseTime(createdAt); err != nil {
			return nil, err
		}
		if n.AcknowledgedAt, err = parseOptionalTime(acked); err != nil {
			return nil, err
		}
		notes = append(notes, n)
	}
	return notes, rows.Err()
}

func (s *Store) MarkNotificationsDelivered(ctx context.Context, ids []int64, at time.Time) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin delivery marks: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	for _, id := range ids {
		if _, err := tx.ExecContext(ctx,
			`UPDATE notifications SET delivered_at = ? WHERE id = ? AND delivered_at = ''`, formatTime(at), id); err != nil {
			return fmt.Errorf("mark notification %d delivered: %w", id, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit delivery marks: %w", err)
	}
	return nil
}
