package service

import (
	"context"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/NexzaDev/argus-asm/internal/domain"
	"github.com/NexzaDev/argus-asm/internal/importer"
)

const maxSourceLength = 128

type FindingView struct {
	domain.Finding
	Risk domain.RiskAssessment `json:"risk"`
}

type FindingFilter struct {
	Status          string
	Severity        string
	Asset           string
	IncludeResolved bool
	Limit           int
}

type FindingDetail struct {
	Finding  FindingView            `json:"finding"`
	Tickets  []domain.Ticket        `json:"tickets"`
	Services []domain.ServiceRecord `json:"services"`
}

type ImportReport struct {
	Format    string              `json:"format"`
	Source    string              `json:"source"`
	DryRun    bool                `json:"dry_run"`
	Accepted  int                 `json:"accepted"`
	Created   int                 `json:"created"`
	Updated   int                 `json:"updated"`
	RowErrors []importer.RowError `json:"row_errors,omitempty"`
}

func (s *Service) ImportFindings(ctx context.Context, actor domain.Actor, format, source string, r io.Reader, dryRun bool) (ImportReport, error) {
	if err := actor.Require(domain.PermOperate); err != nil {
		return ImportReport{}, err
	}
	name := strings.TrimSpace(source)
	if name == "" || len(name) > maxSourceLength {
		return ImportReport{}, fmt.Errorf("%w: source must be 1-%d characters", domain.ErrValidation, maxSourceLength)
	}
	var parsed importer.Result
	var err error
	switch format {
	case "csv":
		parsed, err = importer.ParseFindings(r, importer.DefaultLimits)
	case "json":
		parsed, err = importer.ParseFindingsJSON(r, importer.DefaultLimits)
	case "sarif":
		parsed, err = importer.ParseFindingsSARIF(r, importer.DefaultLimits)
	default:
		return ImportReport{}, fmt.Errorf("%w: unknown format %q (expected csv, json, or sarif)", domain.ErrValidation, format)
	}
	if err != nil {
		return ImportReport{}, err
	}
	report := ImportReport{Format: format, Source: name, DryRun: dryRun, RowErrors: parsed.RowErrors}
	if len(parsed.RowErrors) > 0 {
		return report, fmt.Errorf("%w: %d row(s) rejected; nothing was imported", domain.ErrValidation, len(parsed.RowErrors))
	}
	assets, err := s.store.ListAssets(ctx)
	if err != nil {
		return report, err
	}
	index := indexAssets(assets)
	now := s.now()
	var observations []domain.Observation
	var notifications []domain.Notification
	for _, row := range parsed.Rows {
		kind, value, err := domain.ParseAsset(row.Asset)
		if err != nil {
			report.RowErrors = append(report.RowErrors, importer.RowError{Line: row.Line, Message: err.Error()})
			continue
		}
		asset, ok := index[string(kind)+" "+value]
		if !ok {
			report.RowErrors = append(report.RowErrors, importer.RowError{Line: row.Line,
				Message: fmt.Sprintf("asset %s %s is not in the inventory; add it first", kind, value)})
			continue
		}
		key := domain.DedupeKey(kind, value, row.Title, row.Source)
		observations = append(observations, domain.Observation{
			DedupeKey: key,
			AssetID:   asset.ID,
			Title:     row.Title,
			Source:    row.Source,
			Severity:  row.Severity,
		})
		if row.Severity == domain.SeverityCritical || row.Severity == domain.SeverityHigh {
			notifications = append(notifications, domain.Notification{
				CreatedAt: now,
				Kind:      domain.NotificationFinding,
				Message:   fmt.Sprintf("%s finding on %s: %s (%s)", row.Severity, value, row.Title, row.Source),
				DedupeKey: "finding:" + key + ":" + string(row.Severity),
			})
		}
	}
	if len(report.RowErrors) > 0 {
		return report, fmt.Errorf("%w: %d row(s) rejected; nothing was imported", domain.ErrValidation, len(report.RowErrors))
	}
	report.Accepted = len(observations)
	if dryRun {
		return report, nil
	}
	result, err := s.store.ApplyImport(ctx, domain.ImportBatch{Format: format, Source: name}, observations, notifications, now)
	if err != nil {
		return report, err
	}
	report.Created = result.Counts.Created
	report.Updated = result.Counts.Updated
	detail := fmt.Sprintf("%s source=%s accepted=%d created=%d updated=%d", format, name, report.Accepted, report.Created, report.Updated)
	if err := s.audit(ctx, actor, "findings.import", detail); err != nil {
		return report, err
	}
	return report, nil
}

func indexAssets(assets []domain.Asset) map[string]domain.Asset {
	index := make(map[string]domain.Asset, len(assets))
	for _, asset := range assets {
		index[string(asset.Kind)+" "+asset.Value] = asset
	}
	return index
}

func (s *Service) exposure(ctx context.Context, cache map[int64]int, assetID int64) (int, error) {
	if count, ok := cache[assetID]; ok {
		return count, nil
	}
	services, err := s.store.ListServices(ctx, assetID)
	if err != nil {
		return 0, err
	}
	cache[assetID] = len(services)
	return len(services), nil
}

func (s *Service) view(ctx context.Context, cache map[int64]int, f domain.Finding) (FindingView, error) {
	openServices, err := s.exposure(ctx, cache, f.AssetID)
	if err != nil {
		return FindingView{}, err
	}
	days := 0
	if f.Status == domain.FindingOpen {
		days = int(s.now().Sub(f.FirstSeen).Hours() / 24)
	}
	risk := domain.AssessRisk(f.Severity, f.Status, f.StatusReason, domain.ExposureContext{
		OpenServices: openServices,
		DaysOpen:     days,
	})
	return FindingView{Finding: f, Risk: risk}, nil
}

func (s *Service) ListFindings(ctx context.Context, actor domain.Actor, filter FindingFilter) ([]FindingView, error) {
	if err := actor.Require(domain.PermRead); err != nil {
		return nil, err
	}
	return s.findingViews(ctx, filter)
}

func (s *Service) findingViews(ctx context.Context, filter FindingFilter) ([]FindingView, error) {
	findings, err := s.store.ListFindings(ctx)
	if err != nil {
		return nil, err
	}
	var statusFilter domain.FindingStatus
	if filter.Status != "" {
		if statusFilter, err = domain.ParseFindingStatus(filter.Status); err != nil {
			return nil, err
		}
	}
	var severityFilter domain.Severity
	if filter.Severity != "" {
		if severityFilter, err = domain.ParseSeverity(filter.Severity); err != nil {
			return nil, err
		}
	}
	cache := map[int64]int{}
	views := []FindingView{}
	for _, f := range findings {
		if statusFilter != "" && f.Status != statusFilter {
			continue
		}
		if statusFilter == "" && !filter.IncludeResolved && f.Status == domain.FindingResolved {
			continue
		}
		if severityFilter != "" && f.Severity != severityFilter {
			continue
		}
		if filter.Asset != "" && !strings.Contains(f.AssetValue, strings.ToLower(strings.TrimSpace(filter.Asset))) {
			continue
		}
		view, err := s.view(ctx, cache, f)
		if err != nil {
			return nil, err
		}
		views = append(views, view)
	}
	sort.SliceStable(views, func(i, j int) bool {
		if views[i].Risk.Score != views[j].Risk.Score {
			return views[i].Risk.Score > views[j].Risk.Score
		}
		return views[i].ID < views[j].ID
	})
	if filter.Limit > 0 && len(views) > filter.Limit {
		views = views[:filter.Limit]
	}
	return views, nil
}

func (s *Service) ShowFinding(ctx context.Context, actor domain.Actor, id int64) (FindingDetail, error) {
	if err := actor.Require(domain.PermRead); err != nil {
		return FindingDetail{}, err
	}
	f, err := s.store.FindFinding(ctx, id)
	if err != nil {
		return FindingDetail{}, err
	}
	view, err := s.view(ctx, map[int64]int{}, f)
	if err != nil {
		return FindingDetail{}, err
	}
	tickets, err := s.store.ListTickets(ctx)
	if err != nil {
		return FindingDetail{}, err
	}
	detail := FindingDetail{Finding: view, Tickets: []domain.Ticket{}}
	for _, t := range tickets {
		if t.FindingID == id {
			detail.Tickets = append(detail.Tickets, t)
		}
	}
	services, err := s.store.ListServices(ctx, f.AssetID)
	if err != nil {
		return FindingDetail{}, err
	}
	detail.Services = services
	return detail, nil
}

func (s *Service) SetFindingStatus(ctx context.Context, actor domain.Actor, id int64, statusRaw, reason string) (domain.Finding, error) {
	if err := actor.Require(domain.PermOperate); err != nil {
		return domain.Finding{}, err
	}
	target, err := domain.ParseFindingStatus(statusRaw)
	if err != nil {
		return domain.Finding{}, err
	}
	current, err := s.store.FindFinding(ctx, id)
	if err != nil {
		return domain.Finding{}, err
	}
	trimmed, err := domain.ValidateFindingTransition(current.Status, target, reason)
	if err != nil {
		return domain.Finding{}, err
	}
	if err := s.store.SetFindingStatus(ctx, id, target, trimmed, s.now()); err != nil {
		return domain.Finding{}, err
	}
	detail := fmt.Sprintf("finding=%d %s->%s", id, current.Status, target)
	if trimmed != "" {
		detail += fmt.Sprintf(" reason=%q", trimmed)
	}
	if err := s.audit(ctx, actor, "finding.status", detail); err != nil {
		return domain.Finding{}, err
	}
	return s.store.FindFinding(ctx, id)
}

func (s *Service) ExportFindings(ctx context.Context, actor domain.Actor, includeResolved bool) ([]FindingView, error) {
	if err := actor.Require(domain.PermOperate); err != nil {
		return nil, err
	}
	return s.findingViews(ctx, FindingFilter{IncludeResolved: includeResolved})
}

func (s *Service) ExportAssets(ctx context.Context, actor domain.Actor) ([]domain.Asset, error) {
	if err := actor.Require(domain.PermOperate); err != nil {
		return nil, err
	}
	return s.store.ListAssets(ctx)
}

func (s *Service) Batches(ctx context.Context, actor domain.Actor, source string) ([]domain.ImportBatch, error) {
	if err := actor.Require(domain.PermRead); err != nil {
		return nil, err
	}
	return s.store.ListBatches(ctx, strings.TrimSpace(source))
}
