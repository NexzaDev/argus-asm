package service

import (
	"context"
	"fmt"

	"github.com/NexzaDev/argus-asm/internal/domain"
)

func (s *Service) CreateTicket(ctx context.Context, actor domain.Actor, findingID int64, owner, dueRaw string) (domain.Ticket, error) {
	if err := actor.Require(domain.PermOperate); err != nil {
		return domain.Ticket{}, err
	}
	finding, err := s.store.FindFinding(ctx, findingID)
	if err != nil {
		return domain.Ticket{}, err
	}
	ownerName, err := domain.ValidateOwner(owner)
	if err != nil {
		return domain.Ticket{}, err
	}
	due, err := domain.ParseDueDate(dueRaw)
	if err != nil {
		return domain.Ticket{}, err
	}
	now := s.now()
	created, err := s.store.CreateTicket(ctx, domain.Ticket{
		FindingID:    findingID,
		FindingTitle: finding.Title,
		AssetValue:   finding.AssetValue,
		Owner:        ownerName,
		Status:       domain.TicketOpen,
		DueDate:      due,
		CreatedAt:    now,
		UpdatedAt:    now,
	})
	if err != nil {
		return domain.Ticket{}, err
	}
	detail := fmt.Sprintf("ticket=%d finding=%d owner=%s due=%s", created.ID, findingID, ownerName, due.Format("2006-01-02"))
	if err := s.audit(ctx, actor, "ticket.create", detail); err != nil {
		return domain.Ticket{}, err
	}
	return created, nil
}

func (s *Service) ListTickets(ctx context.Context, actor domain.Actor, overdueOnly bool) ([]domain.Ticket, error) {
	if err := actor.Require(domain.PermRead); err != nil {
		return nil, err
	}
	tickets, err := s.store.ListTickets(ctx)
	if err != nil {
		return nil, err
	}
	if !overdueOnly {
		return tickets, nil
	}
	now := s.now()
	filtered := []domain.Ticket{}
	for _, t := range tickets {
		if t.Overdue(now) {
			filtered = append(filtered, t)
		}
	}
	return filtered, nil
}

func (s *Service) SetTicketStatus(ctx context.Context, actor domain.Actor, id int64, statusRaw string) (domain.Ticket, error) {
	if err := actor.Require(domain.PermOperate); err != nil {
		return domain.Ticket{}, err
	}
	target, err := domain.ParseTicketStatus(statusRaw)
	if err != nil {
		return domain.Ticket{}, err
	}
	current, err := s.store.FindTicket(ctx, id)
	if err != nil {
		return domain.Ticket{}, err
	}
	if err := domain.ValidateTicketTransition(current.Status, target); err != nil {
		return domain.Ticket{}, err
	}
	if err := s.store.SetTicketStatus(ctx, id, target, s.now()); err != nil {
		return domain.Ticket{}, err
	}
	if err := s.audit(ctx, actor, "ticket.status", fmt.Sprintf("ticket=%d %s->%s", id, current.Status, target)); err != nil {
		return domain.Ticket{}, err
	}
	return s.store.FindTicket(ctx, id)
}
