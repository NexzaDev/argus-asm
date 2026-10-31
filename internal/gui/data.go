//go:build !nogui

package gui

import (
	"context"
	"errors"
	"net/netip"
	"strings"
	"time"

	"github.com/NexzaDev/argus-asm/internal/desktop"
	"github.com/NexzaDev/argus-asm/internal/domain"
	"github.com/NexzaDev/argus-asm/internal/repository"
	"github.com/NexzaDev/argus-asm/internal/repository/sqlite"
	"github.com/NexzaDev/argus-asm/internal/service"
)

var errNoDatabase = errors.New("no database yet: run 'argus-asm init' in a terminal, then reopen the dashboard")

type snapshot struct {
	Now           time.Time
	Demo          bool
	Scopes        []domain.Scope
	Assets        []domain.Asset
	Findings      []service.FindingView
	Batches       []domain.ImportBatch
	Notifications []domain.Notification
	Users         []domain.User
	Audit         []domain.AuditEntry
	Verify        domain.AuditVerification
	Inventory     domain.Inventory
}

func withService(ctx context.Context, req desktop.Request, fn func(svc *service.Service, actor domain.Actor) error) error {
	store, err := sqlite.Open(ctx, req.DBPath, false)
	if errors.Is(err, repository.ErrNotInitialized) {
		return errNoDatabase
	}
	if err != nil {
		return err
	}
	defer store.Close()
	now := req.Now
	if now == nil {
		now = time.Now
	}
	svc := service.New(store, now)
	actor, err := svc.ResolveActor(ctx, req.Actor)
	if err != nil {
		return err
	}
	return fn(svc, actor)
}

func loadSnapshot(ctx context.Context, req desktop.Request) (snapshot, error) {
	var snap snapshot
	err := withService(ctx, req, func(svc *service.Service, actor domain.Actor) error {
		var err error
		if snap.Scopes, err = svc.ListScopes(ctx, actor); err != nil {
			return err
		}
		if snap.Assets, err = svc.ListAssets(ctx, actor); err != nil {
			return err
		}
		if snap.Findings, err = svc.ListFindings(ctx, actor, service.FindingFilter{IncludeResolved: true}); err != nil {
			return err
		}
		if snap.Batches, err = svc.Batches(ctx, actor, ""); err != nil {
			return err
		}
		if snap.Notifications, err = svc.Notifications(ctx, actor, true); err != nil {
			return err
		}
		if snap.Users, err = svc.ListUsers(ctx, actor); err != nil {
			return err
		}
		if snap.Audit, snap.Verify, err = svc.AuditLog(ctx, actor); err != nil {
			return err
		}
		snap.Inventory, err = svc.Inventory(ctx, actor)
		return err
	})
	if err != nil {
		return snapshot{}, err
	}
	snap.Now = requestNow(req)
	for _, asset := range snap.Assets {
		if isDemoValue(asset.Value) {
			snap.Demo = true
		}
	}
	return snap, nil
}

func requestNow(req desktop.Request) time.Time {
	if req.Now == nil {
		return time.Now()
	}
	return req.Now()
}

var demoPrefixes = []netip.Prefix{
	netip.MustParsePrefix("192.0.2.0/24"),
	netip.MustParsePrefix("198.51.100.0/24"),
	netip.MustParsePrefix("203.0.113.0/24"),
}

func isDemoValue(value string) bool {
	name := strings.TrimSuffix(strings.ToLower(value), ".")
	if name == "example.com" || strings.HasSuffix(name, ".example.com") ||
		strings.HasSuffix(name, ".test") || strings.HasSuffix(name, ".example") || strings.HasSuffix(name, ".invalid") {
		return true
	}
	addr, err := netip.ParseAddr(name)
	if err != nil {
		return false
	}
	for _, prefix := range demoPrefixes {
		if prefix.Contains(addr) {
			return true
		}
	}
	return false
}

func openFindings(snap snapshot) []service.FindingView {
	var open []service.FindingView
	for _, f := range snap.Findings {
		if f.Status == domain.FindingOpen {
			open = append(open, f)
		}
	}
	return open
}
