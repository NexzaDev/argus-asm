package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/NexzaDev/argus-asm/internal/domain"
)

func (s *Store) UpsertService(ctx context.Context, rec domain.ServiceRecord, seenAt time.Time) (bool, error) {
	stamp := formatTime(seenAt)
	var id int64
	err := s.db.QueryRowContext(ctx,
		`SELECT id FROM services WHERE asset_id = ? AND port = ? AND protocol = ?`,
		rec.AssetID, rec.Port, rec.Protocol).Scan(&id)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		if _, err := s.db.ExecContext(ctx,
			`INSERT INTO services (asset_id, port, protocol, name, source, first_seen, last_seen) VALUES (?, ?, ?, ?, ?, ?, ?)`,
			rec.AssetID, rec.Port, rec.Protocol, rec.Name, rec.Source, stamp, stamp); err != nil {
			return false, fmt.Errorf("insert service: %w", err)
		}
		return true, nil
	case err != nil:
		return false, fmt.Errorf("look up service: %w", err)
	}
	if _, err := s.db.ExecContext(ctx,
		`UPDATE services SET name = CASE WHEN ? <> '' THEN ? ELSE name END, source = ?, last_seen = ? WHERE id = ?`,
		rec.Name, rec.Name, rec.Source, stamp, id); err != nil {
		return false, fmt.Errorf("update service: %w", err)
	}
	return false, nil
}

func (s *Store) ListServices(ctx context.Context, assetID int64) ([]domain.ServiceRecord, error) {
	query := `SELECT sv.id, sv.asset_id, a.value, sv.port, sv.protocol, sv.name, sv.source, sv.first_seen, sv.last_seen
		FROM services sv JOIN assets a ON a.id = sv.asset_id`
	args := []any{}
	if assetID != 0 {
		query += ` WHERE sv.asset_id = ?`
		args = append(args, assetID)
	}
	query += ` ORDER BY a.value, sv.port, sv.protocol`
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list services: %w", err)
	}
	defer rows.Close()
	records := []domain.ServiceRecord{}
	for rows.Next() {
		var rec domain.ServiceRecord
		var firstSeen, lastSeen string
		if err := rows.Scan(&rec.ID, &rec.AssetID, &rec.AssetValue, &rec.Port, &rec.Protocol, &rec.Name, &rec.Source, &firstSeen, &lastSeen); err != nil {
			return nil, fmt.Errorf("scan service: %w", err)
		}
		if rec.FirstSeen, err = parseTime(firstSeen); err != nil {
			return nil, err
		}
		if rec.LastSeen, err = parseTime(lastSeen); err != nil {
			return nil, err
		}
		records = append(records, rec)
	}
	return records, rows.Err()
}

func (s *Store) UpsertCertificate(ctx context.Context, cert domain.Certificate) (bool, error) {
	res, err := s.db.ExecContext(ctx,
		`INSERT OR IGNORE INTO certificates (asset_id, subject, issuer, not_after, fingerprint, created_at) VALUES (?, ?, ?, ?, ?, ?)`,
		cert.AssetID, cert.Subject, cert.Issuer, formatTime(cert.NotAfter), cert.Fingerprint, formatTime(cert.CreatedAt))
	if err != nil {
		return false, fmt.Errorf("insert certificate: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("read certificate insert result: %w", err)
	}
	return n == 1, nil
}

func (s *Store) ListCertificates(ctx context.Context) ([]domain.Certificate, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT c.id, c.asset_id, a.value, c.subject, c.issuer, c.not_after, c.fingerprint, c.created_at
		FROM certificates c JOIN assets a ON a.id = c.asset_id
		ORDER BY c.not_after, c.id`)
	if err != nil {
		return nil, fmt.Errorf("list certificates: %w", err)
	}
	defer rows.Close()
	certs := []domain.Certificate{}
	for rows.Next() {
		var c domain.Certificate
		var notAfter, createdAt string
		if err := rows.Scan(&c.ID, &c.AssetID, &c.AssetValue, &c.Subject, &c.Issuer, &notAfter, &c.Fingerprint, &createdAt); err != nil {
			return nil, fmt.Errorf("scan certificate: %w", err)
		}
		if c.NotAfter, err = parseTime(notAfter); err != nil {
			return nil, err
		}
		if c.CreatedAt, err = parseTime(createdAt); err != nil {
			return nil, err
		}
		certs = append(certs, c)
	}
	return certs, rows.Err()
}

func (s *Store) CreateTicket(ctx context.Context, t domain.Ticket) (domain.Ticket, error) {
	stamp := formatTime(t.CreatedAt)
	res, err := s.db.ExecContext(ctx,
		`INSERT INTO tickets (finding_id, owner, status, due_date, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`,
		t.FindingID, t.Owner, string(t.Status), t.DueDate.Format("2006-01-02"), stamp, stamp)
	if err != nil {
		return domain.Ticket{}, fmt.Errorf("insert ticket: %w", err)
	}
	if t.ID, err = res.LastInsertId(); err != nil {
		return domain.Ticket{}, fmt.Errorf("read ticket id: %w", err)
	}
	return t, nil
}

const ticketSelect = `SELECT t.id, t.finding_id, f.title, a.value, t.owner, t.status, t.due_date, t.created_at, t.updated_at
	FROM tickets t JOIN findings f ON f.id = t.finding_id JOIN assets a ON a.id = f.asset_id`

func scanTicket(row rowScanner) (domain.Ticket, error) {
	var t domain.Ticket
	var status, due, createdAt, updatedAt string
	if err := row.Scan(&t.ID, &t.FindingID, &t.FindingTitle, &t.AssetValue, &t.Owner, &status, &due, &createdAt, &updatedAt); err != nil {
		return domain.Ticket{}, err
	}
	t.Status = domain.TicketStatus(status)
	var err error
	if t.DueDate, err = time.Parse("2006-01-02", due); err != nil {
		return domain.Ticket{}, fmt.Errorf("corrupt due date %q: %w", due, err)
	}
	if t.CreatedAt, err = parseTime(createdAt); err != nil {
		return domain.Ticket{}, err
	}
	if t.UpdatedAt, err = parseTime(updatedAt); err != nil {
		return domain.Ticket{}, err
	}
	return t, nil
}

func (s *Store) ListTickets(ctx context.Context) ([]domain.Ticket, error) {
	rows, err := s.db.QueryContext(ctx, ticketSelect+` ORDER BY t.due_date, t.id`)
	if err != nil {
		return nil, fmt.Errorf("list tickets: %w", err)
	}
	defer rows.Close()
	tickets := []domain.Ticket{}
	for rows.Next() {
		t, err := scanTicket(rows)
		if err != nil {
			return nil, fmt.Errorf("scan ticket: %w", err)
		}
		tickets = append(tickets, t)
	}
	return tickets, rows.Err()
}

func (s *Store) FindTicket(ctx context.Context, id int64) (domain.Ticket, error) {
	t, err := scanTicket(s.db.QueryRowContext(ctx, ticketSelect+` WHERE t.id = ?`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Ticket{}, fmt.Errorf("%w: ticket %d", domain.ErrNotFound, id)
	}
	return t, err
}

func (s *Store) SetTicketStatus(ctx context.Context, id int64, status domain.TicketStatus, at time.Time) error {
	res, err := s.db.ExecContext(ctx, `UPDATE tickets SET status = ?, updated_at = ? WHERE id = ?`,
		string(status), formatTime(at), id)
	if err != nil {
		return fmt.Errorf("update ticket: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return fmt.Errorf("%w: ticket %d", domain.ErrNotFound, id)
	}
	return nil
}
