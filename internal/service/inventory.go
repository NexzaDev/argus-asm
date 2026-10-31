package service

import (
	"context"
	"fmt"
	"time"

	"github.com/NexzaDev/argus-asm/internal/domain"
)

func (s *Service) AddScope(ctx context.Context, actor domain.Actor, raw string) (domain.Scope, error) {
	if err := actor.Require(domain.PermAdminister); err != nil {
		return domain.Scope{}, err
	}
	scope, err := domain.NewScope(raw)
	if err != nil {
		return domain.Scope{}, err
	}
	scope.CreatedAt = s.now()
	created, err := s.store.CreateScope(ctx, scope)
	if err != nil {
		return domain.Scope{}, err
	}
	if err := s.audit(ctx, actor, "scope.add", fmt.Sprintf("%s %s", created.Kind, created.Value)); err != nil {
		return domain.Scope{}, err
	}
	return created, nil
}

func (s *Service) ListScopes(ctx context.Context, actor domain.Actor) ([]domain.Scope, error) {
	if err := actor.Require(domain.PermRead); err != nil {
		return nil, err
	}
	return s.store.ListScopes(ctx)
}

func (s *Service) AddAsset(ctx context.Context, actor domain.Actor, raw string) (domain.Asset, error) {
	if err := actor.Require(domain.PermOperate); err != nil {
		return domain.Asset{}, err
	}
	kind, value, err := domain.ParseAsset(raw)
	if err != nil {
		return domain.Asset{}, err
	}
	scopes, err := s.store.ListScopes(ctx)
	if err != nil {
		return domain.Asset{}, err
	}
	scopeID, err := authorizingScope(scopes, kind, value)
	if err != nil {
		return domain.Asset{}, err
	}
	created, err := s.store.CreateAsset(ctx, domain.Asset{
		ScopeID:   scopeID,
		Kind:      kind,
		Value:     value,
		CreatedAt: s.now(),
	})
	if err != nil {
		return domain.Asset{}, err
	}
	if err := s.audit(ctx, actor, "asset.add", fmt.Sprintf("%s %s", kind, value)); err != nil {
		return domain.Asset{}, err
	}
	return created, nil
}

func authorizingScope(scopes []domain.Scope, kind domain.AssetKind, value string) (int64, error) {
	for _, scope := range scopes {
		if scope.Authorizes(kind, value) {
			return scope.ID, nil
		}
	}
	return 0, fmt.Errorf("%w: %s %s is not inside any registered scope; add a scope first", domain.ErrOutOfScope, kind, value)
}

func (s *Service) ListAssets(ctx context.Context, actor domain.Actor) ([]domain.Asset, error) {
	if err := actor.Require(domain.PermRead); err != nil {
		return nil, err
	}
	return s.store.ListAssets(ctx)
}

func (s *Service) resolveAsset(ctx context.Context, raw string) (domain.Asset, error) {
	kind, value, err := domain.ParseAsset(raw)
	if err != nil {
		return domain.Asset{}, err
	}
	return s.store.FindAsset(ctx, kind, value)
}

func (s *Service) ShowAsset(ctx context.Context, actor domain.Actor, raw string) (domain.Asset, error) {
	if err := actor.Require(domain.PermRead); err != nil {
		return domain.Asset{}, err
	}
	return s.resolveAsset(ctx, raw)
}

func (s *Service) RemoveAsset(ctx context.Context, actor domain.Actor, raw string, dryRun bool) (domain.Asset, domain.AssetImpact, error) {
	if err := actor.Require(domain.PermAdminister); err != nil {
		return domain.Asset{}, domain.AssetImpact{}, err
	}
	asset, err := s.resolveAsset(ctx, raw)
	if err != nil {
		return domain.Asset{}, domain.AssetImpact{}, err
	}
	impact, err := s.store.AssetImpact(ctx, asset.ID)
	if err != nil {
		return domain.Asset{}, domain.AssetImpact{}, err
	}
	if dryRun {
		return asset, impact, nil
	}
	if err := s.store.DeleteAsset(ctx, asset.ID); err != nil {
		return domain.Asset{}, domain.AssetImpact{}, err
	}
	detail := fmt.Sprintf("%s %s findings=%d services=%d certificates=%d relations=%d tickets=%d",
		asset.Kind, asset.Value, impact.Findings, impact.Services, impact.Certificates, impact.Relations, impact.Tickets)
	if err := s.audit(ctx, actor, "asset.remove", detail); err != nil {
		return domain.Asset{}, domain.AssetImpact{}, err
	}
	return asset, impact, nil
}

func (s *Service) SetOwnership(ctx context.Context, actor domain.Actor, raw, owner, verifiedBy string) (domain.Asset, error) {
	if err := actor.Require(domain.PermOperate); err != nil {
		return domain.Asset{}, err
	}
	asset, err := s.resolveAsset(ctx, raw)
	if err != nil {
		return domain.Asset{}, err
	}
	ownerName, err := domain.ValidateOwner(owner)
	if err != nil {
		return domain.Asset{}, err
	}
	var verifiedAt *time.Time
	verifier := ""
	if verifiedBy != "" {
		if verifier, err = domain.ValidateOwner(verifiedBy); err != nil {
			return domain.Asset{}, err
		}
		now := s.now()
		verifiedAt = &now
	}
	if err := s.store.UpdateAssetOwnership(ctx, asset.ID, ownerName, verifier, verifiedAt); err != nil {
		return domain.Asset{}, err
	}
	if err := s.audit(ctx, actor, "asset.owner", fmt.Sprintf("%s %s owner=%s verified_by=%s", asset.Kind, asset.Value, ownerName, verifier)); err != nil {
		return domain.Asset{}, err
	}
	return s.store.FindAsset(ctx, asset.Kind, asset.Value)
}

func (s *Service) SetProbeAllowed(ctx context.Context, actor domain.Actor, raw string, allowed bool) (domain.Asset, error) {
	if err := actor.Require(domain.PermAdminister); err != nil {
		return domain.Asset{}, err
	}
	asset, err := s.resolveAsset(ctx, raw)
	if err != nil {
		return domain.Asset{}, err
	}
	if err := s.store.SetAssetProbeAllowed(ctx, asset.ID, allowed); err != nil {
		return domain.Asset{}, err
	}
	if err := s.audit(ctx, actor, "asset.probe_policy", fmt.Sprintf("%s %s allowed=%t", asset.Kind, asset.Value, allowed)); err != nil {
		return domain.Asset{}, err
	}
	return s.store.FindAsset(ctx, asset.Kind, asset.Value)
}

func (s *Service) Link(ctx context.Context, actor domain.Actor, fromRaw, toRaw, kindRaw string) (domain.Relation, error) {
	if err := actor.Require(domain.PermOperate); err != nil {
		return domain.Relation{}, err
	}
	kind, err := domain.ParseRelationKind(kindRaw)
	if err != nil {
		return domain.Relation{}, err
	}
	from, err := s.resolveAsset(ctx, fromRaw)
	if err != nil {
		return domain.Relation{}, err
	}
	to, err := s.resolveAsset(ctx, toRaw)
	if err != nil {
		return domain.Relation{}, err
	}
	if from.ID == to.ID {
		return domain.Relation{}, fmt.Errorf("%w: an asset cannot be related to itself", domain.ErrValidation)
	}
	created, err := s.store.CreateRelation(ctx, domain.Relation{
		FromID:    from.ID,
		FromValue: from.Value,
		ToID:      to.ID,
		ToValue:   to.Value,
		Kind:      kind,
		CreatedAt: s.now(),
	})
	if err != nil {
		return domain.Relation{}, err
	}
	if err := s.audit(ctx, actor, "relation.add", fmt.Sprintf("%s %s %s", from.Value, kind, to.Value)); err != nil {
		return domain.Relation{}, err
	}
	return created, nil
}

func (s *Service) Relations(ctx context.Context, actor domain.Actor, raw string) ([]domain.Relation, error) {
	if err := actor.Require(domain.PermRead); err != nil {
		return nil, err
	}
	asset, err := s.resolveAsset(ctx, raw)
	if err != nil {
		return nil, err
	}
	return s.store.ListRelations(ctx, asset.ID)
}
