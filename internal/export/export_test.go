package export

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/NexzaDev/argus-asm/internal/domain"
	"github.com/NexzaDev/argus-asm/internal/service"
)

func TestNeutralizeFormulaPrefixes(t *testing.T) {
	for _, value := range []string{"=HYPERLINK(\"x\")", "+1", "-2", "@cmd", "\tcell", "\rcell"} {
		if got := Neutralize(value); !strings.HasPrefix(got, "'") {
			t.Errorf("Neutralize(%q) = %q, want leading apostrophe", value, got)
		}
	}
	for _, value := range []string{"plain", "", "192.0.2.10"} {
		if got := Neutralize(value); got != value {
			t.Errorf("Neutralize(%q) = %q, want unchanged", value, got)
		}
	}
}

func TestWriteFindingsCSVNeutralizesUntrustedCells(t *testing.T) {
	var buf bytes.Buffer
	views := []service.FindingView{{
		Finding: domain.Finding{
			ID:         7,
			AssetValue: "www.example.test",
			Title:      "=cmd|' /C calc'!A0",
			Source:     "scanner",
			Severity:   domain.SeverityHigh,
			Status:     domain.FindingOpen,
			FirstSeen:  time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC),
			LastSeen:   time.Date(2026, 10, 2, 0, 0, 0, 0, time.UTC),
		},
	}}
	if err := WriteFindingsCSV(&buf, views); err != nil {
		t.Fatalf("WriteFindingsCSV() error = %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "'=cmd") {
		t.Fatalf("formula not neutralized in output:\n%s", out)
	}
	if !strings.HasPrefix(out, "id,risk_score,severity,status,asset,title,source,status_reason,first_seen,last_seen") {
		t.Fatalf("unexpected header:\n%s", out)
	}
}
