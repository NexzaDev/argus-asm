package importer

import (
	"errors"
	"strings"
	"testing"

	"github.com/NexzaDev/argus-asm/internal/domain"
)

const header = "asset,title,severity,source\n"

func parse(t *testing.T, input string, limits Limits) (Result, error) {
	t.Helper()
	return ParseFindings(strings.NewReader(input), limits)
}

func TestParseFindingsValid(t *testing.T) {
	result, err := parse(t, header+"www.example.test,Expired certificate,HIGH,scanner\n", DefaultLimits)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if len(result.RowErrors) != 0 || len(result.Rows) != 1 {
		t.Fatalf("result = %+v", result)
	}
	row := result.Rows[0]
	if row.Line != 2 || row.Asset != "www.example.test" || row.Severity != domain.SeverityHigh || row.Source != "scanner" {
		t.Fatalf("row = %+v", row)
	}
}

func TestParseFindingsHandlesBOMCRLFAndQuotes(t *testing.T) {
	input := "\ufeffSource , ASSET,Title,Severity\r\nscanner,192.0.2.1,\"Open port 22, SSH\",low\r\n"
	result, err := parse(t, input, DefaultLimits)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if len(result.Rows) != 1 || result.Rows[0].Title != "Open port 22, SSH" || result.Rows[0].Asset != "192.0.2.1" {
		t.Fatalf("result = %+v", result)
	}
}

func TestParseFindingsCollectsRowErrors(t *testing.T) {
	input := header + "a.example.test,,low,s\nb.example.test,T,nope,s\n,T,low,s\nc.example.test,T,low\nd.example.test,T,low,s\n"
	result, err := parse(t, input, DefaultLimits)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if len(result.Rows) != 1 || result.Rows[0].Asset != "d.example.test" {
		t.Fatalf("valid rows = %+v", result.Rows)
	}
	if len(result.RowErrors) != 4 {
		t.Fatalf("row errors = %+v", result.RowErrors)
	}
	wantLines := []int{2, 3, 4, 5}
	for i, rowErr := range result.RowErrors {
		if rowErr.Line != wantLines[i] {
			t.Fatalf("row error %d line = %d; want %d", i, rowErr.Line, wantLines[i])
		}
	}
}

func TestParseFindingsHeaderProblems(t *testing.T) {
	cases := map[string]string{
		"missing column": "asset,title,severity\n",
		"duplicate":      "asset,asset,title,severity,source\n",
		"empty":          "",
	}
	for name, input := range cases {
		if _, err := parse(t, input, DefaultLimits); !errors.Is(err, domain.ErrValidation) {
			t.Fatalf("%s: error = %v; want ErrValidation", name, err)
		}
	}
}

func TestParseFindingsEnforcesLimits(t *testing.T) {
	rows := header + "a.example.test,T,low,s\nb.example.test,T,low,s\n"
	if _, err := parse(t, rows, Limits{MaxBytes: 1 << 20, MaxRows: 1, MaxFieldLength: 512}); !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("row limit error = %v", err)
	}
	if _, err := parse(t, rows, Limits{MaxBytes: 20, MaxRows: 10, MaxFieldLength: 512}); !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("byte limit error = %v", err)
	}
	long := header + "a.example.test," + strings.Repeat("x", 20) + ",low,s\n"
	result, err := parse(t, long, Limits{MaxBytes: 1 << 20, MaxRows: 10, MaxFieldLength: 10})
	if err != nil || len(result.RowErrors) != 1 {
		t.Fatalf("field length result = %+v, %v", result, err)
	}
}

func TestParseFindingsAcceptsExactByteLimit(t *testing.T) {
	input := header + "a.example.test,T,low,s\n"
	if _, err := parse(t, input, Limits{MaxBytes: int64(len(input)), MaxRows: 10, MaxFieldLength: 512}); err != nil {
		t.Fatalf("file exactly at limit rejected: %v", err)
	}
}

func TestParseFindingsRejectsMalformedQuoting(t *testing.T) {
	input := header + "a.example.test,\"unterminated,low,s\n"
	if _, err := parse(t, input, DefaultLimits); !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("malformed CSV error = %v", err)
	}
}

func FuzzParseFindings(f *testing.F) {
	f.Add(header + "a.example.test,T,low,s\n")
	f.Add("asset,title,severity,source\n\"x,y\",\"\"\"q\"\"\",high,s\n")
	f.Add("\xff\xfe,,,\n")
	f.Fuzz(func(t *testing.T, input string) {
		result, err := ParseFindings(strings.NewReader(input), DefaultLimits)
		if err != nil {
			return
		}
		for _, row := range result.Rows {
			if row.Asset == "" || row.Title == "" || row.Source == "" {
				t.Fatalf("accepted row with empty required field: %+v", row)
			}
			if _, err := domain.ParseSeverity(string(row.Severity)); err != nil {
				t.Fatalf("accepted unknown severity %q", row.Severity)
			}
		}
	})
}
