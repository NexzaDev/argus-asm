package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"time"
)

type Severity string

const (
	SeverityCritical Severity = "critical"
	SeverityHigh     Severity = "high"
	SeverityMedium   Severity = "medium"
	SeverityLow      Severity = "low"
	SeverityInfo     Severity = "info"
)

func ParseSeverity(raw string) (Severity, error) {
	switch s := Severity(strings.ToLower(strings.TrimSpace(raw))); s {
	case SeverityCritical, SeverityHigh, SeverityMedium, SeverityLow, SeverityInfo:
		return s, nil
	}
	return "", validationErrorf("unknown severity %q (expected critical, high, medium, low, or info)", raw)
}

func (s Severity) Weight() int {
	switch s {
	case SeverityCritical:
		return 90
	case SeverityHigh:
		return 70
	case SeverityMedium:
		return 40
	case SeverityLow:
		return 15
	case SeverityInfo:
		return 5
	default:
		return 0
	}
}

type Finding struct {
	ID              int64         `json:"id"`
	AssetID         int64         `json:"asset_id"`
	AssetKind       AssetKind     `json:"asset_kind"`
	AssetValue      string        `json:"asset"`
	Title           string        `json:"title"`
	Source          string        `json:"source"`
	Severity        Severity      `json:"severity"`
	Status          FindingStatus `json:"status"`
	StatusReason    string        `json:"status_reason,omitempty"`
	StatusUpdatedAt time.Time     `json:"status_updated_at"`
	FirstSeen       time.Time     `json:"first_seen"`
	LastSeen        time.Time     `json:"last_seen"`
}

type Observation struct {
	DedupeKey string
	AssetID   int64
	Title     string
	Source    string
	Severity  Severity
}

type BatchObservation struct {
	FindingID  int64
	AssetValue string
	Title      string
	Source     string
	Severity   Severity
}

type ImportCounts struct {
	Created int
	Updated int
}

type ImportBatch struct {
	ID        int64     `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	Format    string    `json:"format"`
	Source    string    `json:"source"`
	Accepted  int       `json:"accepted"`
	Created   int       `json:"created"`
	Updated   int       `json:"updated"`
}

type ImportResult struct {
	Batch  ImportBatch
	Counts ImportCounts
}

func DedupeKey(kind AssetKind, value, title, source string) string {
	parts := []string{
		string(kind),
		value,
		strings.ToLower(strings.TrimSpace(title)),
		strings.ToLower(strings.TrimSpace(source)),
	}
	sum := sha256.Sum256([]byte(strings.Join(parts, "\x1f")))
	return hex.EncodeToString(sum[:])
}
