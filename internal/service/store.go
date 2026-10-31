package service

import (
	"context"
	"time"

	"github.com/NexzaDev/argus-asm/internal/domain"
)

type Store interface {
	SchemaVersion(ctx context.Context) (int, error)
	Inventory(ctx context.Context) (domain.Inventory, error)
	CreateScope(ctx context.Context, scope domain.Scope) (domain.Scope, error)
	ListScopes(ctx context.Context) ([]domain.Scope, error)
	CreateAsset(ctx context.Context, asset domain.Asset) (domain.Asset, error)
	ListAssets(ctx context.Context) ([]domain.Asset, error)
	FindAsset(ctx context.Context, kind domain.AssetKind, value string) (domain.Asset, error)
	UpdateAssetOwnership(ctx context.Context, id int64, owner, verifiedBy string, verifiedAt *time.Time) error
	SetAssetProbeAllowed(ctx context.Context, id int64, allowed bool) error
	AssetImpact(ctx context.Context, id int64) (domain.AssetImpact, error)
	DeleteAsset(ctx context.Context, id int64) error
	CreateRelation(ctx context.Context, rel domain.Relation) (domain.Relation, error)
	ListRelations(ctx context.Context, assetID int64) ([]domain.Relation, error)
	ApplyImport(ctx context.Context, batch domain.ImportBatch, observations []domain.Observation, notifications []domain.Notification, seenAt time.Time) (domain.ImportResult, error)
	ListFindings(ctx context.Context) ([]domain.Finding, error)
	FindFinding(ctx context.Context, id int64) (domain.Finding, error)
	SetFindingStatus(ctx context.Context, id int64, status domain.FindingStatus, reason string, at time.Time) error
	ListBatches(ctx context.Context, source string) ([]domain.ImportBatch, error)
	BatchObservations(ctx context.Context, batchID int64) ([]domain.BatchObservation, error)
	UpsertService(ctx context.Context, rec domain.ServiceRecord, seenAt time.Time) (bool, error)
	ListServices(ctx context.Context, assetID int64) ([]domain.ServiceRecord, error)
	UpsertCertificate(ctx context.Context, cert domain.Certificate) (bool, error)
	ListCertificates(ctx context.Context) ([]domain.Certificate, error)
	CreateTicket(ctx context.Context, ticket domain.Ticket) (domain.Ticket, error)
	ListTickets(ctx context.Context) ([]domain.Ticket, error)
	FindTicket(ctx context.Context, id int64) (domain.Ticket, error)
	SetTicketStatus(ctx context.Context, id int64, status domain.TicketStatus, at time.Time) error
	CreateNotifications(ctx context.Context, notifications []domain.Notification) (int, error)
	ListNotifications(ctx context.Context, includeAcknowledged bool) ([]domain.Notification, error)
	AcknowledgeNotification(ctx context.Context, id int64, at time.Time) error
	UndeliveredNotifications(ctx context.Context) ([]domain.Notification, error)
	Snapshot(ctx context.Context) ([]byte, error)
	MarkNotificationsDelivered(ctx context.Context, ids []int64, at time.Time) error
	CreateUser(ctx context.Context, user domain.User) (domain.User, error)
	FindUser(ctx context.Context, name string) (domain.User, error)
	ListUsers(ctx context.Context) ([]domain.User, error)
	SetUserRole(ctx context.Context, name string, role domain.Role) error
	CountAdmins(ctx context.Context) (int, error)
	AppendAudit(ctx context.Context, entry domain.AuditEntry) error
	ListAudit(ctx context.Context) ([]domain.AuditEntry, error)
}
