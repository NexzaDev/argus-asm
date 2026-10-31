package importer

import (
	"strings"
	"testing"

	"github.com/NexzaDev/argus-asm/internal/domain"
)

const sarifSample = `{
  "version": "2.1.0",
  "runs": [{
    "tool": {"driver": {"name": "demo-sarif"}},
    "results": [
      {"ruleId": "TLS-EXPIRED", "level": "error",
       "message": {"text": "certificate expired"},
       "locations": [{"physicalLocation": {"artifactLocation": {"uri": "https://www.example.test/login"}}}],
       "properties": {"security-severity": "7.5"}},
      {"message": {"text": "SSH exposed"}, "level": "warning",
       "locations": [{"physicalLocation": {"artifactLocation": {"uri": "192.0.2.10"}}}]},
      {"ruleId": "INFO-BANNER", "level": "note",
       "locations": [{"physicalLocation": {"artifactLocation": {"uri": "api.example.test"}}}],
       "properties": {"security-severity": 0}}
    ]
  }]
}`

func TestParseFindingsSARIFMapsResults(t *testing.T) {
	result, err := ParseFindingsSARIF(strings.NewReader(sarifSample), DefaultLimits)
	if err != nil {
		t.Fatalf("ParseFindingsSARIF() error = %v", err)
	}
	if len(result.RowErrors) != 0 || len(result.Rows) != 3 {
		t.Fatalf("rows = %+v, row errors = %+v", result.Rows, result.RowErrors)
	}
	first := result.Rows[0]
	if first.Asset != "www.example.test" || first.Title != "TLS-EXPIRED" || first.Source != "demo-sarif" || first.Severity != domain.SeverityHigh {
		t.Fatalf("first row = %+v", first)
	}
	second := result.Rows[1]
	if second.Title != "SSH exposed" || second.Severity != domain.SeverityMedium || second.Asset != "192.0.2.10" || second.Line != 2 {
		t.Fatalf("second row = %+v", second)
	}
	if result.Rows[2].Severity != domain.SeverityInfo {
		t.Fatalf("third row severity = %s, want info (security-severity 0)", result.Rows[2].Severity)
	}
}

func TestParseFindingsSARIFSeverityBands(t *testing.T) {
	cases := map[string]domain.Severity{
		"9.8": domain.SeverityCritical, "9.0": domain.SeverityCritical, "8.9": domain.SeverityHigh,
		"7.0": domain.SeverityHigh, "6.9": domain.SeverityMedium, "4.0": domain.SeverityMedium,
		"3.9": domain.SeverityLow, "0.1": domain.SeverityLow, "0": domain.SeverityInfo,
	}
	for score, want := range cases {
		input := `{"version":"2.1.0","runs":[{"tool":{"driver":{"name":"s"}},"results":[{"ruleId":"R","level":"error",
		"locations":[{"physicalLocation":{"artifactLocation":{"uri":"192.0.2.1"}}}],"properties":{"security-severity":"` + score + `"}}]}]}`
		result, err := ParseFindingsSARIF(strings.NewReader(input), DefaultLimits)
		if err != nil || len(result.Rows) != 1 {
			t.Fatalf("score %s: err = %v, rows = %+v, row errors = %+v", score, err, result.Rows, result.RowErrors)
		}
		if result.Rows[0].Severity != want {
			t.Fatalf("score %s: severity = %s, want %s", score, result.Rows[0].Severity, want)
		}
	}
}

func TestParseFindingsSARIFLevelFallback(t *testing.T) {
	levels := map[string]domain.Severity{"error": domain.SeverityHigh, "": domain.SeverityMedium, "warning": domain.SeverityMedium, "note": domain.SeverityLow, "none": domain.SeverityInfo}
	for level, want := range levels {
		input := `{"version":"2.1.0","runs":[{"tool":{"driver":{"name":"s"}},"results":[{"ruleId":"R","level":"` + level + `",
		"locations":[{"physicalLocation":{"artifactLocation":{"uri":"192.0.2.1"}}}]}]}]}`
		result, err := ParseFindingsSARIF(strings.NewReader(input), DefaultLimits)
		if err != nil || len(result.Rows) != 1 || result.Rows[0].Severity != want {
			t.Fatalf("level %q: err = %v, result = %+v", level, err, result)
		}
	}
}

func TestParseFindingsSARIFReportsRowErrors(t *testing.T) {
	input := `{"version":"2.1.0","runs":[{"tool":{"driver":{"name":"s"}},"results":[
	{"ruleId":"R","level":"error","message":{"text":"no location"}},
	{"ruleId":"R","level":"error","locations":[{"physicalLocation":{"artifactLocation":{"uri":"src/main.go"}}}]},
	{"ruleId":"R","level":"bogus","locations":[{"physicalLocation":{"artifactLocation":{"uri":"192.0.2.1"}}}]}
	]}]}`
	result, err := ParseFindingsSARIF(strings.NewReader(input), DefaultLimits)
	if err != nil {
		t.Fatalf("ParseFindingsSARIF() error = %v", err)
	}
	if len(result.Rows) != 1 || result.Rows[0].Line != 2 || len(result.RowErrors) != 2 {
		t.Fatalf("rows = %+v, row errors = %+v", result.Rows, result.RowErrors)
	}
	if result.RowErrors[0].Line != 1 || result.RowErrors[1].Line != 3 {
		t.Fatalf("row error lines = %+v", result.RowErrors)
	}
}

func TestParseFindingsSARIFRejectsOtherVersionsAndMalformedJSON(t *testing.T) {
	for name, input := range map[string]string{
		"version 2.0.0": `{"version":"2.0.0","runs":[]}`,
		"no version":    `{"runs":[]}`,
		"not json":      `{"version":`,
		"array":         `[]`,
	} {
		if _, err := ParseFindingsSARIF(strings.NewReader(input), DefaultLimits); err == nil {
			t.Fatalf("%s: expected an error", name)
		}
	}
}

func TestParseFindingsSARIFEnforcesRowLimit(t *testing.T) {
	limits := Limits{MaxBytes: 1 << 20, MaxRows: 1, MaxFieldLength: 512}
	input := `{"version":"2.1.0","runs":[{"tool":{"driver":{"name":"s"}},"results":[
	{"ruleId":"A","level":"note","locations":[{"physicalLocation":{"artifactLocation":{"uri":"192.0.2.1"}}}]},
	{"ruleId":"B","level":"note","locations":[{"physicalLocation":{"artifactLocation":{"uri":"192.0.2.1"}}}]}]}]}`
	if _, err := ParseFindingsSARIF(strings.NewReader(input), limits); err == nil {
		t.Fatal("expected a row-limit error")
	}
}

func TestParseFindingsSARIFEnforcesByteLimit(t *testing.T) {
	limits := Limits{MaxBytes: 20, MaxRows: 10, MaxFieldLength: 512}
	if _, err := ParseFindingsSARIF(strings.NewReader(sarifSample), limits); err == nil {
		t.Fatal("expected a byte-limit error")
	}
}
