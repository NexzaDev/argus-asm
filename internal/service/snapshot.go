package service

import (
	"context"

	"github.com/NexzaDev/argus-asm/internal/domain"
)

func (s *Service) SnapshotDatabase(ctx context.Context, actor domain.Actor) ([]byte, error) {
	if err := actor.Require(domain.PermAdminister); err != nil {
		return nil, err
	}
	data, err := s.store.Snapshot(ctx)
	if err != nil {
		return nil, err
	}
	if err := s.audit(ctx, actor, "db.snapshot", "snapshot taken"); err != nil {
		return nil, err
	}
	return data, nil
}
