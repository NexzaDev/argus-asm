package service

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/NexzaDev/argus-asm/internal/domain"
	"github.com/NexzaDev/argus-asm/internal/importer"
)

const (
	maxServiceNameLength = 128
	certExpiryWindowDays = 30
)

type ServiceImportReport struct {
	DryRun    bool                `json:"dry_run"`
	Accepted  int                 `json:"accepted"`
	Created   int                 `json:"created"`
	Updated   int                 `json:"updated"`
	RowErrors []importer.RowError `json:"row_errors,omitempty"`
}

type CertificateImportReport struct {
	DryRun     bool   `json:"dry_run"`
	Asset      string `json:"asset"`
	Accepted   int    `json:"accepted"`
	Created    int    `json:"created"`
	Duplicates int    `json:"duplicates"`
	Expiring   int    `json:"expiring_within_30_days"`
}

func (s *Service) AddService(ctx context.Context, actor domain.Actor, assetRaw, portRaw, name string) (domain.ServiceRecord, bool, error) {
	if err := actor.Require(domain.PermOperate); err != nil {
		return domain.ServiceRecord{}, false, err
	}
	asset, err := s.resolveAsset(ctx, assetRaw)
	if err != nil {
		return domain.ServiceRecord{}, false, err
	}
	port, protocol, err := domain.ParsePortSpec(portRaw)
	if err != nil {
		return domain.ServiceRecord{}, false, err
	}
	cleaned := strings.TrimSpace(name)
	if len(cleaned) > maxServiceNameLength {
		return domain.ServiceRecord{}, false, fmt.Errorf("%w: service name exceeds %d characters", domain.ErrValidation, maxServiceNameLength)
	}
	rec := domain.ServiceRecord{AssetID: asset.ID, AssetValue: asset.Value, Port: port, Protocol: protocol, Name: cleaned, Source: "manual"}
	created, err := s.store.UpsertService(ctx, rec, s.now())
	if err != nil {
		return domain.ServiceRecord{}, false, err
	}
	if err := s.audit(ctx, actor, "service.add", fmt.Sprintf("%s %d/%s created=%t", asset.Value, port, protocol, created)); err != nil {
		return domain.ServiceRecord{}, false, err
	}
	return rec, created, nil
}

func (s *Service) ListServices(ctx context.Context, actor domain.Actor, assetRaw string) ([]domain.ServiceRecord, error) {
	if err := actor.Require(domain.PermRead); err != nil {
		return nil, err
	}
	var assetID int64
	if strings.TrimSpace(assetRaw) != "" {
		asset, err := s.resolveAsset(ctx, assetRaw)
		if err != nil {
			return nil, err
		}
		assetID = asset.ID
	}
	return s.store.ListServices(ctx, assetID)
}

func (s *Service) ImportNmap(ctx context.Context, actor domain.Actor, r io.Reader, dryRun bool) (ServiceImportReport, error) {
	if err := actor.Require(domain.PermOperate); err != nil {
		return ServiceImportReport{}, err
	}
	parsed, err := importer.ParseNmapXML(r, importer.DefaultLimits)
	if err != nil {
		return ServiceImportReport{}, err
	}
	report := ServiceImportReport{DryRun: dryRun, RowErrors: parsed.RowErrors}
	assets, err := s.store.ListAssets(ctx)
	if err != nil {
		return report, err
	}
	index := indexAssets(assets)
	var records []domain.ServiceRecord
	for _, svc := range parsed.Services {
		asset, ok := matchNmapTarget(index, svc.Targets)
		if !ok {
			report.RowErrors = append(report.RowErrors, importer.RowError{Line: svc.Host,
				Message: fmt.Sprintf("no inventory asset matches %s; add it first", strings.Join(svc.Targets, ", "))})
			continue
		}
		records = append(records, domain.ServiceRecord{
			AssetID:    asset.ID,
			AssetValue: asset.Value,
			Port:       svc.Port,
			Protocol:   svc.Protocol,
			Name:       svc.Name,
			Source:     "nmap",
		})
	}
	if len(report.RowErrors) > 0 {
		return report, fmt.Errorf("%w: %d row(s) rejected; nothing was imported", domain.ErrValidation, len(report.RowErrors))
	}
	report.Accepted = len(records)
	if dryRun {
		return report, nil
	}
	now := s.now()
	for _, rec := range records {
		created, err := s.store.UpsertService(ctx, rec, now)
		if err != nil {
			return report, err
		}
		if created {
			report.Created++
		} else {
			report.Updated++
		}
	}
	detail := fmt.Sprintf("nmap accepted=%d created=%d updated=%d", report.Accepted, report.Created, report.Updated)
	if err := s.audit(ctx, actor, "services.import", detail); err != nil {
		return report, err
	}
	return report, nil
}

func matchNmapTarget(index map[string]domain.Asset, targets []string) (domain.Asset, bool) {
	for _, target := range targets {
		kind, value, err := domain.ParseAsset(target)
		if err != nil {
			continue
		}
		if asset, ok := index[string(kind)+" "+value]; ok {
			return asset, true
		}
	}
	return domain.Asset{}, false
}

func (s *Service) ImportCertificates(ctx context.Context, actor domain.Actor, assetRaw string, data []byte, dryRun bool) (CertificateImportReport, error) {
	if err := actor.Require(domain.PermOperate); err != nil {
		return CertificateImportReport{}, err
	}
	asset, err := s.resolveAsset(ctx, assetRaw)
	if err != nil {
		return CertificateImportReport{}, err
	}
	parsed, err := importer.ParseCertificatesPEM(data, importer.DefaultLimits)
	if err != nil {
		return CertificateImportReport{}, err
	}
	now := s.now()
	report := CertificateImportReport{DryRun: dryRun, Asset: asset.Value, Accepted: len(parsed)}
	var notifications []domain.Notification
	for _, c := range parsed {
		cert := domain.Certificate{
			AssetID:     asset.ID,
			AssetValue:  asset.Value,
			Subject:     c.Subject,
			Issuer:      c.Issuer,
			NotAfter:    c.NotAfter,
			Fingerprint: c.Fingerprint,
			CreatedAt:   now,
		}
		if cert.ExpiresWithin(now, certExpiryWindowDays) {
			report.Expiring++
			notifications = append(notifications, domain.Notification{
				CreatedAt: now,
				Kind:      domain.NotificationCertificate,
				Message:   fmt.Sprintf("certificate %s on %s expires %s", shortFingerprint(c.Fingerprint), asset.Value, c.NotAfter.Format("2006-01-02")),
				DedupeKey: "cert:" + c.Fingerprint,
			})
		}
		if dryRun {
			continue
		}
		inserted, err := s.store.UpsertCertificate(ctx, cert)
		if err != nil {
			return report, err
		}
		if inserted {
			report.Created++
		} else {
			report.Duplicates++
		}
	}
	if dryRun {
		return report, nil
	}
	if len(notifications) > 0 {
		if _, err := s.store.CreateNotifications(ctx, notifications); err != nil {
			return report, err
		}
	}
	detail := fmt.Sprintf("asset=%s accepted=%d created=%d duplicates=%d", asset.Value, report.Accepted, report.Created, report.Duplicates)
	if err := s.audit(ctx, actor, "certificates.import", detail); err != nil {
		return report, err
	}
	return report, nil
}

func shortFingerprint(fingerprint string) string {
	if len(fingerprint) > 12 {
		return fingerprint[:12]
	}
	return fingerprint
}

func (s *Service) ListCertificates(ctx context.Context, actor domain.Actor) ([]domain.Certificate, error) {
	if err := actor.Require(domain.PermRead); err != nil {
		return nil, err
	}
	return s.store.ListCertificates(ctx)
}
