package export

import (
	"encoding/csv"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/NexzaDev/argus-asm/internal/domain"
	"github.com/NexzaDev/argus-asm/internal/service"
)

const formulaPrefixes = "=+-@\t\r"

func Neutralize(value string) string {
	if value != "" && strings.ContainsRune(formulaPrefixes, rune(value[0])) {
		return "'" + value
	}
	return value
}

func WriteFindingsCSV(w io.Writer, views []service.FindingView) error {
	writer := csv.NewWriter(w)
	if err := writer.Write([]string{"id", "risk_score", "severity", "status", "asset", "title", "source", "status_reason", "first_seen", "last_seen"}); err != nil {
		return fmt.Errorf("write findings CSV: %w", err)
	}
	for _, v := range views {
		record := []string{
			strconv.FormatInt(v.ID, 10),
			strconv.Itoa(v.Risk.Score),
			string(v.Severity),
			string(v.Status),
			Neutralize(v.AssetValue),
			Neutralize(v.Title),
			Neutralize(v.Source),
			Neutralize(v.StatusReason),
			v.FirstSeen.UTC().Format("2006-01-02T15:04:05Z"),
			v.LastSeen.UTC().Format("2006-01-02T15:04:05Z"),
		}
		if err := writer.Write(record); err != nil {
			return fmt.Errorf("write findings CSV: %w", err)
		}
	}
	writer.Flush()
	return writer.Error()
}

func WriteAssetsCSV(w io.Writer, assets []domain.Asset) error {
	writer := csv.NewWriter(w)
	if err := writer.Write([]string{"id", "kind", "value", "owner", "verified_by", "probe_allowed", "created_at"}); err != nil {
		return fmt.Errorf("write assets CSV: %w", err)
	}
	for _, a := range assets {
		record := []string{
			strconv.FormatInt(a.ID, 10),
			string(a.Kind),
			Neutralize(a.Value),
			Neutralize(a.OwnerName),
			Neutralize(a.VerifiedBy),
			strconv.FormatBool(a.ProbeAllowed),
			a.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
		}
		if err := writer.Write(record); err != nil {
			return fmt.Errorf("write assets CSV: %w", err)
		}
	}
	writer.Flush()
	return writer.Error()
}
