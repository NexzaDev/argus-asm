package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/NexzaDev/argus-asm/internal/domain"
)

func (s *Store) CreateScope(ctx context.Context, scope domain.Scope) (domain.Scope, error) {
	result, err := s.db.ExecContext(ctx,
		`INSERT INTO scopes (kind, value, created_at) VALUES (?, ?, ?)`,
		string(scope.Kind), scope.Value, formatTime(scope.CreatedAt))
	if err != nil {
		if isUniqueViolation(err) {
			return domain.Scope{}, fmt.Errorf("%w: scope %s", domain.ErrDuplicate, scope.Value)
		}
		return domain.Scope{}, fmt.Errorf("insert scope: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return domain.Scope{}, fmt.Errorf("read scope id: %w", err)
	}
	scope.ID = id
	return scope, nil
}

func (s *Store) ListScopes(ctx context.Context) ([]domain.Scope, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, kind, value, created_at FROM scopes ORDER BY kind, value`)
	if err != nil {
		return nil, fmt.Errorf("list scopes: %w", err)
	}
	defer rows.Close()
	scopes := []domain.Scope{}
	for rows.Next() {
		var scope domain.Scope
		var kind, createdAt string
		if err := rows.Scan(&scope.ID, &kind, &scope.Value, &createdAt); err != nil {
			return nil, fmt.Errorf("scan scope: %w", err)
		}
		scope.Kind = domain.ScopeKind(kind)
		if scope.CreatedAt, err = parseTime(createdAt); err != nil {
			return nil, err
		}
		scopes = append(scopes, scope)
	}
	return scopes, rows.Err()
}

const assetColumns = `a.id, a.scope_id, a.kind, a.value, a.owner_name, a.verified_by, a.verified_at, a.probe_allowed, a.created_at`

func scanAsset(row rowScanner) (domain.Asset, error) {
	var asset domain.Asset
	var kind, verifiedAt, createdAt string
	var probe int
	if err := row.Scan(&asset.ID, &asset.ScopeID, &kind, &asset.Value, &asset.OwnerName,
		&asset.VerifiedBy, &verifiedAt, &probe, &createdAt); err != nil {
		return domain.Asset{}, err
	}
	asset.Kind = domain.AssetKind(kind)
	asset.ProbeAllowed = probe == 1
	var err error
	if asset.VerifiedAt, err = parseOptionalTime(verifiedAt); err != nil {
		return domain.Asset{}, err
	}
	if asset.CreatedAt, err = parseTime(createdAt); err != nil {
		return domain.Asset{}, err
	}
	return asset, nil
}

func (s *Store) CreateAsset(ctx context.Context, asset domain.Asset) (domain.Asset, error) {
	result, err := s.db.ExecContext(ctx,
		`INSERT INTO assets (scope_id, kind, value, created_at) VALUES (?, ?, ?, ?)`,
		asset.ScopeID, string(asset.Kind), asset.Value, formatTime(asset.CreatedAt))
	if err != nil {
		if isUniqueViolation(err) {
			return domain.Asset{}, fmt.Errorf("%w: asset %s %s", domain.ErrDuplicate, asset.Kind, asset.Value)
		}
		return domain.Asset{}, fmt.Errorf("insert asset: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return domain.Asset{}, fmt.Errorf("read asset id: %w", err)
	}
	asset.ID = id
	return asset, nil
}

func (s *Store) ListAssets(ctx context.Context) ([]domain.Asset, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT `+assetColumns+` FROM assets a ORDER BY a.kind, a.value`)
	if err != nil {
		return nil, fmt.Errorf("list assets: %w", err)
	}
	defer rows.Close()
	assets := []domain.Asset{}
	for rows.Next() {
		asset, err := scanAsset(rows)
		if err != nil {
			return nil, fmt.Errorf("scan asset: %w", err)
		}
		assets = append(assets, asset)
	}
	return assets, rows.Err()
}

func (s *Store) FindAsset(ctx context.Context, kind domain.AssetKind, value string) (domain.Asset, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT `+assetColumns+` FROM assets a WHERE a.kind = ? AND a.value = ?`, string(kind), value)
	asset, err := scanAsset(row)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Asset{}, fmt.Errorf("%w: asset %s %s", domain.ErrNotFound, kind, value)
	}
	return asset, err
}

func (s *Store) UpdateAssetOwnership(ctx context.Context, id int64, owner, verifiedBy string, verifiedAt *time.Time) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE assets SET owner_name = ?, verified_by = ?, verified_at = ? WHERE id = ?`,
		owner, verifiedBy, formatOptionalTime(verifiedAt), id)
	if err != nil {
		return fmt.Errorf("update asset ownership: %w", err)
	}
	return nil
}

func (s *Store) SetAssetProbeAllowed(ctx context.Context, id int64, allowed bool) error {
	value := 0
	if allowed {
		value = 1
	}
	if _, err := s.db.ExecContext(ctx, `UPDATE assets SET probe_allowed = ? WHERE id = ?`, value, id); err != nil {
		return fmt.Errorf("update probe permission: %w", err)
	}
	return nil
}

func (s *Store) AssetImpact(ctx context.Context, id int64) (domain.AssetImpact, error) {
	var impact domain.AssetImpact
	counts := []struct {
		query  string
		target *int
	}{
		{`SELECT COUNT(*) FROM findings WHERE asset_id = ?`, &impact.Findings},
		{`SELECT COUNT(*) FROM services WHERE asset_id = ?`, &impact.Services},
		{`SELECT COUNT(*) FROM certificates WHERE asset_id = ?`, &impact.Certificates},
		{`SELECT COUNT(*) FROM tickets t JOIN findings f ON f.id = t.finding_id WHERE f.asset_id = ?`, &impact.Tickets},
	}
	for _, c := range counts {
		if err := s.db.QueryRowContext(ctx, c.query, id).Scan(c.target); err != nil {
			return domain.AssetImpact{}, fmt.Errorf("count asset impact: %w", err)
		}
	}
	if err := s.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM asset_relations WHERE from_asset_id = ? OR to_asset_id = ?`, id, id,
	).Scan(&impact.Relations); err != nil {
		return domain.AssetImpact{}, fmt.Errorf("count asset relations: %w", err)
	}
	return impact, nil
}

func (s *Store) DeleteAsset(ctx context.Context, id int64) error {
	result, err := s.db.ExecContext(ctx, `DELETE FROM assets WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete asset: %w", err)
	}
	if n, _ := result.RowsAffected(); n == 0 {
		return fmt.Errorf("%w: asset id %d", domain.ErrNotFound, id)
	}
	return nil
}

func (s *Store) CreateRelation(ctx context.Context, rel domain.Relation) (domain.Relation, error) {
	result, err := s.db.ExecContext(ctx,
		`INSERT INTO asset_relations (from_asset_id, to_asset_id, kind, created_at) VALUES (?, ?, ?, ?)`,
		rel.FromID, rel.ToID, string(rel.Kind), formatTime(rel.CreatedAt))
	if err != nil {
		if isUniqueViolation(err) {
			return domain.Relation{}, fmt.Errorf("%w: relation %s", domain.ErrDuplicate, rel.Kind)
		}
		return domain.Relation{}, fmt.Errorf("insert relation: %w", err)
	}
	rel.ID, err = result.LastInsertId()
	if err != nil {
		return domain.Relation{}, fmt.Errorf("read relation id: %w", err)
	}
	return rel, nil
}

func (s *Store) ListRelations(ctx context.Context, assetID int64) ([]domain.Relation, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT r.id, r.from_asset_id, fa.value, r.to_asset_id, ta.value, r.kind, r.created_at
		FROM asset_relations r
		JOIN assets fa ON fa.id = r.from_asset_id
		JOIN assets ta ON ta.id = r.to_asset_id
		WHERE r.from_asset_id = ? OR r.to_asset_id = ?
		ORDER BY r.id`, assetID, assetID)
	if err != nil {
		return nil, fmt.Errorf("list relations: %w", err)
	}
	defer rows.Close()
	relations := []domain.Relation{}
	for rows.Next() {
		var rel domain.Relation
		var kind, createdAt string
		if err := rows.Scan(&rel.ID, &rel.FromID, &rel.FromValue, &rel.ToID, &rel.ToValue, &kind, &createdAt); err != nil {
			return nil, fmt.Errorf("scan relation: %w", err)
		}
		rel.Kind = domain.RelationKind(kind)
		if rel.CreatedAt, err = parseTime(createdAt); err != nil {
			return nil, err
		}
		relations = append(relations, rel)
	}
	return relations, rows.Err()
}
