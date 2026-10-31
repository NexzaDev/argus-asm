package service

import (
	"context"
	"fmt"
	"io"

	"github.com/NexzaDev/argus-asm/internal/domain"
	"github.com/NexzaDev/argus-asm/internal/outbox"
)

func (s *Service) Notifications(ctx context.Context, actor domain.Actor, includeAcknowledged bool) ([]domain.Notification, error) {
	if err := actor.Require(domain.PermRead); err != nil {
		return nil, err
	}
	return s.store.ListNotifications(ctx, includeAcknowledged)
}

func (s *Service) Acknowledge(ctx context.Context, actor domain.Actor, id int64) error {
	if err := actor.Require(domain.PermOperate); err != nil {
		return err
	}
	if err := s.store.AcknowledgeNotification(ctx, id, s.now()); err != nil {
		return err
	}
	return s.audit(ctx, actor, "notification.ack", fmt.Sprintf("notification=%d", id))
}

func (s *Service) RefreshNotifications(ctx context.Context, actor domain.Actor) (int, error) {
	if err := actor.Require(domain.PermOperate); err != nil {
		return 0, err
	}
	now := s.now()
	tickets, err := s.store.ListTickets(ctx)
	if err != nil {
		return 0, err
	}
	var pending []domain.Notification
	for _, t := range tickets {
		if !t.Overdue(now) {
			continue
		}
		pending = append(pending, domain.Notification{
			CreatedAt: now,
			Kind:      domain.NotificationTicket,
			Message: fmt.Sprintf("ticket %d overdue (owner %s, due %s): %s on %s",
				t.ID, t.Owner, t.DueDate.Format("2006-01-02"), t.FindingTitle, t.AssetValue),
			DedupeKey: fmt.Sprintf("ticket:%d:%s", t.ID, t.DueDate.Format("2006-01-02")),
		})
	}
	certs, err := s.store.ListCertificates(ctx)
	if err != nil {
		return 0, err
	}
	for _, c := range certs {
		if !c.ExpiresWithin(now, certExpiryWindowDays) {
			continue
		}
		pending = append(pending, domain.Notification{
			CreatedAt: now,
			Kind:      domain.NotificationCertificate,
			Message:   fmt.Sprintf("certificate %s on %s expires %s", shortFingerprint(c.Fingerprint), c.AssetValue, c.NotAfter.Format("2006-01-02")),
			DedupeKey: "cert:" + c.Fingerprint,
		})
	}
	created, err := s.store.CreateNotifications(ctx, pending)
	if err != nil {
		return 0, err
	}
	if err := s.audit(ctx, actor, "notifications.refresh", fmt.Sprintf("created=%d", created)); err != nil {
		return 0, err
	}
	return created, nil
}

func (s *Service) PendingNotifications(ctx context.Context, actor domain.Actor) ([]domain.Notification, error) {
	if err := actor.Require(domain.PermOperate); err != nil {
		return nil, err
	}
	return s.store.UndeliveredNotifications(ctx)
}

func (s *Service) DeliverNotifications(ctx context.Context, actor domain.Actor, w io.Writer) (int, error) {
	pending, err := s.PendingNotifications(ctx, actor)
	if err != nil {
		return 0, err
	}
	if len(pending) == 0 {
		return 0, nil
	}
	if err := outbox.Write(w, pending); err != nil {
		return 0, err
	}
	ids := make([]int64, 0, len(pending))
	for _, n := range pending {
		ids = append(ids, n.ID)
	}
	if err := s.store.MarkNotificationsDelivered(ctx, ids, s.now()); err != nil {
		return 0, err
	}
	if err := s.audit(ctx, actor, "notifications.deliver", fmt.Sprintf("delivered=%d", len(ids))); err != nil {
		return 0, err
	}
	return len(ids), nil
}
