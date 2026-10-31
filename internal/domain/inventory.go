package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
)

type Inventory struct {
	Scopes       int `json:"scopes"`
	Assets       int `json:"assets"`
	Findings     int `json:"findings"`
	Services     int `json:"services"`
	Certificates int `json:"certificates"`
	Tickets      int `json:"tickets"`
	Users        int `json:"users"`
}

type AssetImpact struct {
	Findings     int `json:"findings"`
	Services     int `json:"services"`
	Certificates int `json:"certificates"`
	Relations    int `json:"relations"`
	Tickets      int `json:"tickets"`
}

type RelationKind string

const (
	RelationResolvesTo RelationKind = "resolves_to"
	RelationHosts      RelationKind = "hosts"
	RelationRelatedTo  RelationKind = "related_to"
)

func ParseRelationKind(raw string) (RelationKind, error) {
	switch kind := RelationKind(strings.ToLower(strings.TrimSpace(raw))); kind {
	case RelationResolvesTo, RelationHosts, RelationRelatedTo:
		return kind, nil
	}
	return "", validationErrorf("unknown relation kind %q (expected resolves_to, hosts, or related_to)", raw)
}

type Relation struct {
	ID        int64        `json:"id"`
	FromID    int64        `json:"from_id"`
	FromValue string       `json:"from"`
	ToID      int64        `json:"to_id"`
	ToValue   string       `json:"to"`
	Kind      RelationKind `json:"kind"`
	CreatedAt time.Time    `json:"created_at"`
}

const (
	minPort = 1
	maxPort = 65535
)

type ServiceRecord struct {
	ID         int64     `json:"id"`
	AssetID    int64     `json:"asset_id"`
	AssetValue string    `json:"asset"`
	Port       int       `json:"port"`
	Protocol   string    `json:"protocol"`
	Name       string    `json:"name"`
	Source     string    `json:"source"`
	FirstSeen  time.Time `json:"first_seen"`
	LastSeen   time.Time `json:"last_seen"`
}

func ParsePortSpec(raw string) (int, string, error) {
	spec := strings.ToLower(strings.TrimSpace(raw))
	portText, protocol, hasProtocol := strings.Cut(spec, "/")
	if !hasProtocol {
		protocol = "tcp"
	}
	port, err := strconv.Atoi(portText)
	if err != nil || port < minPort || port > maxPort {
		return 0, "", validationErrorf("port %q must be an integer from %d to %d", raw, minPort, maxPort)
	}
	if protocol != "tcp" && protocol != "udp" {
		return 0, "", validationErrorf("protocol %q must be tcp or udp", protocol)
	}
	return port, protocol, nil
}

type Certificate struct {
	ID          int64     `json:"id"`
	AssetID     int64     `json:"asset_id"`
	AssetValue  string    `json:"asset"`
	Subject     string    `json:"subject"`
	Issuer      string    `json:"issuer"`
	NotAfter    time.Time `json:"not_after"`
	Fingerprint string    `json:"fingerprint_sha256"`
	CreatedAt   time.Time `json:"created_at"`
}

func (c Certificate) ExpiresWithin(now time.Time, days int) bool {
	return c.NotAfter.Before(now.Add(time.Duration(days) * 24 * time.Hour))
}

type Ticket struct {
	ID           int64        `json:"id"`
	FindingID    int64        `json:"finding_id"`
	FindingTitle string       `json:"finding_title"`
	AssetValue   string       `json:"asset"`
	Owner        string       `json:"owner"`
	Status       TicketStatus `json:"status"`
	DueDate      time.Time    `json:"due_date"`
	CreatedAt    time.Time    `json:"created_at"`
	UpdatedAt    time.Time    `json:"updated_at"`
}

func (t Ticket) Overdue(now time.Time) bool {
	startOfToday := now.UTC().Truncate(24 * time.Hour)
	return t.Status != TicketDone && t.DueDate.Before(startOfToday)
}

const (
	NotificationFinding     = "finding"
	NotificationCertificate = "certificate"
	NotificationTicket      = "ticket"
)

type Notification struct {
	ID             int64      `json:"id"`
	CreatedAt      time.Time  `json:"created_at"`
	Kind           string     `json:"kind"`
	Message        string     `json:"message"`
	DedupeKey      string     `json:"-"`
	AcknowledgedAt *time.Time `json:"acknowledged_at,omitempty"`
}

type AuditEntry struct {
	ID       int64     `json:"id"`
	At       time.Time `json:"at"`
	Actor    string    `json:"actor"`
	Action   string    `json:"action"`
	Detail   string    `json:"detail"`
	PrevHash string    `json:"prev_hash"`
	Hash     string    `json:"hash"`
}

func AuditHash(prevHash string, at time.Time, actor, action, detail string) string {
	payload, _ := json.Marshal([]string{prevHash, at.UTC().Format(time.RFC3339Nano), actor, action, detail})
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:])
}

type AuditVerification struct {
	OK      bool   `json:"ok"`
	Total   int    `json:"total"`
	Chained int    `json:"chained"`
	Legacy  int    `json:"legacy_unchained"`
	Problem string `json:"problem,omitempty"`
	EntryID int64  `json:"entry_id,omitempty"`
}

func VerifyAuditChain(entries []AuditEntry) AuditVerification {
	result := AuditVerification{Total: len(entries), OK: true}
	prev := ""
	chaining := false
	for _, entry := range entries {
		if entry.Hash == "" {
			if chaining {
				return failVerification(result, entry.ID, "unchained entry follows chained entries")
			}
			result.Legacy++
			continue
		}
		chaining = true
		if entry.PrevHash != prev {
			return failVerification(result, entry.ID, "previous-hash link is broken")
		}
		if AuditHash(entry.PrevHash, entry.At, entry.Actor, entry.Action, entry.Detail) != entry.Hash {
			return failVerification(result, entry.ID, "entry content does not match its hash")
		}
		prev = entry.Hash
		result.Chained++
	}
	return result
}

func failVerification(result AuditVerification, id int64, problem string) AuditVerification {
	result.OK = false
	result.Problem = fmt.Sprintf("entry %d: %s", id, problem)
	result.EntryID = id
	return result
}
