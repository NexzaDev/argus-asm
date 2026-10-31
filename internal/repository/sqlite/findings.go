package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/NexzaDev/argus-asm/internal/domain"
)

func (s *Store) ApplyImport(ctx context.Context, batch domain.ImportBatch, observations []domain.Observation, notifications []domain.Notification, seenAt time.Time) (domain.ImportResult, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.ImportResult{}, fmt.Errorf("begin import: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	stamp := formatTime(seenAt)
	res, err := tx.ExecContext(ctx,
		`INSERT INTO import_batches (created_at, format, source, accepted, created, updated) VALUES (?, ?, ?, ?, 0, 0)`,
		stamp, batch.Format, batch.Source, len(observations))
	if err != nil {
		return domain.ImportResult{}, fmt.Errorf("record import batch: %w", err)
	}
	batchID, err := res.LastInsertId()
	if err != nil {
		return domain.ImportResult{}, fmt.Errorf("read batch id: %w", err)
	}

	var counts domain.ImportCounts
	for _, obs := range observations {
		var id int64
		err := tx.QueryRowContext(ctx, `SELECT id FROM findings WHERE dedupe_key = ?`, obs.DedupeKey).Scan(&id)
		switch {
		case errors.Is(err, sql.ErrNoRows):
			res, err := tx.ExecContext(ctx,
				`INSERT INTO findings (asset_id, title, severity, source, dedupe_key, first_seen, last_seen, status, status_updated_at)
				 VALUES (?, ?, ?, ?, ?, ?, ?, 'open', ?)`,
				obs.AssetID, obs.Title, string(obs.Severity), obs.Source, obs.DedupeKey, stamp, stamp, stamp)
			if err != nil {
				return domain.ImportResult{}, fmt.Errorf("insert finding: %w", err)
			}
			if id, err = res.LastInsertId(); err != nil {
				return domain.ImportResult{}, fmt.Errorf("read finding id: %w", err)
			}
			counts.Created++
		case err != nil:
			return domain.ImportResult{}, fmt.Errorf("look up finding: %w", err)
		default:
			if _, err := tx.ExecContext(ctx,
				`UPDATE findings SET severity = ?, last_seen = ? WHERE id = ?`,
				string(obs.Severity), stamp, id); err != nil {
				return domain.ImportResult{}, fmt.Errorf("update finding: %w", err)
			}
			counts.Updated++
		}
		if _, err := tx.ExecContext(ctx,
			`INSERT OR REPLACE INTO finding_observations (batch_id, finding_id, severity) VALUES (?, ?, ?)`,
			batchID, id, string(obs.Severity)); err != nil {
			return domain.ImportResult{}, fmt.Errorf("record observation: %w", err)
		}
	}

	if _, err := tx.ExecContext(ctx,
		`UPDATE import_batches SET created = ?, updated = ? WHERE id = ?`,
		counts.Created, counts.Updated, batchID); err != nil {
		return domain.ImportResult{}, fmt.Errorf("finalize import batch: %w", err)
	}
	for _, n := range notifications {
		if err := insertNotification(ctx, tx, n); err != nil {
			return domain.ImportResult{}, err
		}
	}
	if err := tx.Commit(); err != nil {
		return domain.ImportResult{}, fmt.Errorf("commit import: %w", err)
	}
	batch.ID = batchID
	batch.Accepted = len(observations)
	batch.Created = counts.Created
	batch.Updated = counts.Updated
	return domain.ImportResult{Batch: batch, Counts: counts}, nil
}

const findingColumns = `f.id, f.asset_id, a.kind, a.value, f.title, f.source, f.severity, f.status, f.status_reason, f.status_updated_at, f.first_seen, f.last_seen`

func scanFinding(row rowScanner) (domain.Finding, error) {
	var f domain.Finding
	var kind, severity, status, updated, firstSeen, lastSeen string
	if err := row.Scan(&f.ID, &f.AssetID, &kind, &f.AssetValue, &f.Title, &f.Source, &severity,
		&status, &f.StatusReason, &updated, &firstSeen, &lastSeen); err != nil {
		return domain.Finding{}, err
	}
	f.AssetKind = domain.AssetKind(kind)
	f.Severity = domain.Severity(severity)
	f.Status = domain.FindingStatus(status)
	var err error
	if f.StatusUpdatedAt, err = parseTime(updated); err != nil {
		return domain.Finding{}, err
	}
	if f.FirstSeen, err = parseTime(firstSeen); err != nil {
		return domain.Finding{}, err
	}
	if f.LastSeen, err = parseTime(lastSeen); err != nil {
		return domain.Finding{}, err
	}
	return f, nil
}

func (s *Store) ListFindings(ctx context.Context) ([]domain.Finding, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT `+findingColumns+` FROM findings f JOIN assets a ON a.id = f.asset_id ORDER BY f.id`)
	if err != nil {
		return nil, fmt.Errorf("list findings: %w", err)
	}
	defer rows.Close()
	findings := []domain.Finding{}
	for rows.Next() {
		f, err := scanFinding(rows)
		if err != nil {
			return nil, fmt.Errorf("scan finding: %w", err)
		}
		findings = append(findings, f)
	}
	return findings, rows.Err()
}

func (s *Store) FindFinding(ctx context.Context, id int64) (domain.Finding, error) {
	f, err := scanFinding(s.db.QueryRowContext(ctx,
		`SELECT `+findingColumns+` FROM findings f JOIN assets a ON a.id = f.asset_id WHERE f.id = ?`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Finding{}, fmt.Errorf("%w: finding %d", domain.ErrNotFound, id)
	}
	return f, err
}

func (s *Store) SetFindingStatus(ctx context.Context, id int64, status domain.FindingStatus, reason string, at time.Time) error {
	res, err := s.db.ExecContext(ctx,
		`UPDATE findings SET status = ?, status_reason = ?, status_updated_at = ? WHERE id = ?`,
		string(status), reason, formatTime(at), id)
	if err != nil {
		return fmt.Errorf("update finding status: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return fmt.Errorf("%w: finding %d", domain.ErrNotFound, id)
	}
	return nil
}

func (s *Store) ListBatches(ctx context.Context, source string) ([]domain.ImportBatch, error) {
	query := `SELECT id, created_at, format, source, accepted, created, updated FROM import_batches`
	args := []any{}
	if source != "" {
		query += ` WHERE source = ?`
		args = append(args, source)
	}
	query += ` ORDER BY id DESC`
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list import batches: %w", err)
	}
	defer rows.Close()
	batches := []domain.ImportBatch{}
	for rows.Next() {
		var b domain.ImportBatch
		var createdAt string
		if err := rows.Scan(&b.ID, &createdAt, &b.Format, &b.Source, &b.Accepted, &b.Created, &b.Updated); err != nil {
			return nil, fmt.Errorf("scan import batch: %w", err)
		}
		if b.CreatedAt, err = parseTime(createdAt); err != nil {
			return nil, err
		}
		batches = append(batches, b)
	}
	return batches, rows.Err()
}

func (s *Store) BatchObservations(ctx context.Context, batchID int64) ([]domain.BatchObservation, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT o.finding_id, a.value, f.title, f.source, o.severity
		FROM finding_observations o
		JOIN findings f ON f.id = o.finding_id
		JOIN assets a ON a.id = f.asset_id
		WHERE o.batch_id = ?
		ORDER BY o.finding_id`, batchID)
	if err != nil {
		return nil, fmt.Errorf("list batch observations: %w", err)
	}
	defer rows.Close()
	observations := []domain.BatchObservation{}
	for rows.Next() {
		var o domain.BatchObservation
		var severity string
		if err := rows.Scan(&o.FindingID, &o.AssetValue, &o.Title, &o.Source, &severity); err != nil {
			return nil, fmt.Errorf("scan batch observation: %w", err)
		}
		o.Severity = domain.Severity(severity)
		observations = append(observations, o)
	}
	return observations, rows.Err()
}
