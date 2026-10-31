package service

import (
	"context"
	"fmt"

	"github.com/NexzaDev/argus-asm/internal/domain"
)

func (s *Service) CreateUser(ctx context.Context, actor domain.Actor, name, roleRaw string) (domain.User, error) {
	if err := actor.Require(domain.PermAdminister); err != nil {
		return domain.User{}, err
	}
	userName, err := domain.ValidateUserName(name)
	if err != nil {
		return domain.User{}, err
	}
	if userName == domain.BootstrapActorName {
		return domain.User{}, fmt.Errorf("%w: %q is reserved for the bootstrap actor", domain.ErrValidation, userName)
	}
	role, err := domain.ParseRole(roleRaw)
	if err != nil {
		return domain.User{}, err
	}
	if actor.Name == domain.BootstrapActorName && role != domain.RoleAdmin {
		return domain.User{}, fmt.Errorf("%w: the first user must have the admin role so that administration remains possible", domain.ErrValidation)
	}
	created, err := s.store.CreateUser(ctx, domain.User{Name: userName, Role: role, CreatedAt: s.now()})
	if err != nil {
		return domain.User{}, err
	}
	if err := s.audit(ctx, actor, "user.add", fmt.Sprintf("%s role=%s", userName, role)); err != nil {
		return domain.User{}, err
	}
	return created, nil
}

func (s *Service) ListUsers(ctx context.Context, actor domain.Actor) ([]domain.User, error) {
	if err := actor.Require(domain.PermRead); err != nil {
		return nil, err
	}
	return s.store.ListUsers(ctx)
}

func (s *Service) SetUserRole(ctx context.Context, actor domain.Actor, name, roleRaw string) (domain.User, error) {
	if err := actor.Require(domain.PermAdminister); err != nil {
		return domain.User{}, err
	}
	role, err := domain.ParseRole(roleRaw)
	if err != nil {
		return domain.User{}, err
	}
	user, err := s.store.FindUser(ctx, name)
	if err != nil {
		return domain.User{}, err
	}
	if user.Role == domain.RoleAdmin && role != domain.RoleAdmin {
		admins, err := s.store.CountAdmins(ctx)
		if err != nil {
			return domain.User{}, err
		}
		if admins <= 1 {
			return domain.User{}, fmt.Errorf("%w: cannot demote the last administrator", domain.ErrValidation)
		}
	}
	if err := s.store.SetUserRole(ctx, user.Name, role); err != nil {
		return domain.User{}, err
	}
	if err := s.audit(ctx, actor, "user.role", fmt.Sprintf("%s %s->%s", user.Name, user.Role, role)); err != nil {
		return domain.User{}, err
	}
	return s.store.FindUser(ctx, user.Name)
}

func (s *Service) AuditLog(ctx context.Context, actor domain.Actor) ([]domain.AuditEntry, domain.AuditVerification, error) {
	if err := actor.Require(domain.PermAdminister); err != nil {
		return nil, domain.AuditVerification{}, err
	}
	entries, err := s.store.ListAudit(ctx)
	if err != nil {
		return nil, domain.AuditVerification{}, err
	}
	return entries, domain.VerifyAuditChain(entries), nil
}
