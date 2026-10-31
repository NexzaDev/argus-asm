package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/NexzaDev/argus-asm/internal/domain"
)

const ActorEnvVar = "ARGUS_ASM_ACTOR"

type Service struct {
	store Store
	now   func() time.Time
}

func New(store Store, now func() time.Time) *Service {
	return &Service{store: store, now: now}
}

func (s *Service) ResolveActor(ctx context.Context, requested string) (domain.Actor, error) {
	users, err := s.store.ListUsers(ctx)
	if err != nil {
		return domain.Actor{}, err
	}
	if len(users) == 0 {
		if requested != "" && requested != domain.BootstrapActorName {
			return domain.Actor{}, fmt.Errorf("%w: no users exist yet; only actor %q is available until one is added", domain.ErrForbidden, domain.BootstrapActorName)
		}
		return domain.Actor{Name: domain.BootstrapActorName, Role: domain.RoleAdmin}, nil
	}
	if requested == "" {
		return domain.Actor{}, fmt.Errorf("%w: an actor is required; pass --actor or set %s", domain.ErrForbidden, ActorEnvVar)
	}
	user, err := s.store.FindUser(ctx, requested)
	if errors.Is(err, domain.ErrNotFound) {
		return domain.Actor{}, fmt.Errorf("%w: unknown actor %q", domain.ErrForbidden, requested)
	}
	if err != nil {
		return domain.Actor{}, err
	}
	return domain.Actor{Name: user.Name, Role: user.Role}, nil
}

func (s *Service) audit(ctx context.Context, actor domain.Actor, action, detail string) error {
	return s.store.AppendAudit(ctx, domain.AuditEntry{
		At:     s.now(),
		Actor:  actor.Name,
		Action: action,
		Detail: detail,
	})
}

func (s *Service) SchemaVersion(ctx context.Context) (int, error) {
	return s.store.SchemaVersion(ctx)
}

func (s *Service) Inventory(ctx context.Context, actor domain.Actor) (domain.Inventory, error) {
	if err := actor.Require(domain.PermRead); err != nil {
		return domain.Inventory{}, err
	}
	return s.store.Inventory(ctx)
}
