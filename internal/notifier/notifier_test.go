package notifier

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/NexzaDev/argus-asm/internal/domain"
)

type fakeSource struct {
	notes []domain.Notification
	err   error
}

func (f *fakeSource) Notifications(context.Context) ([]domain.Notification, error) {
	return f.notes, f.err
}

type fakeSender struct {
	titles, bodies []string
	err            error
}

func (f *fakeSender) Send(_ context.Context, title, body string) error {
	if f.err != nil {
		return f.err
	}
	f.titles = append(f.titles, title)
	f.bodies = append(f.bodies, body)
	return nil
}

func finding(id int64, sev domain.Severity) domain.Notification {
	return domain.Notification{ID: id, Kind: domain.NotificationFinding, Message: string(sev) + " finding on api.example.test: SQL injection (scanner)"}
}

func TestSummarizeMinimalUsesCountsOnly(t *testing.T) {
	_, body := Summarize([]domain.Notification{finding(1, domain.SeverityCritical), finding(2, domain.SeverityCritical)}, DetailMinimal)
	if body != "2 new critical findings in Argus" {
		t.Fatalf("body = %q", body)
	}
	if strings.Contains(body, "api.example.test") || strings.Contains(body, "SQL") {
		t.Fatalf("minimal text leaked details: %q", body)
	}
}

func TestSummarizeSingularAndMixed(t *testing.T) {
	_, one := Summarize([]domain.Notification{finding(1, domain.SeverityHigh)}, DetailMinimal)
	if one != "1 new high finding in Argus" {
		t.Fatalf("singular = %q", one)
	}
	mixed := []domain.Notification{finding(1, domain.SeverityCritical), finding(2, domain.SeverityHigh), {ID: 3, Kind: domain.NotificationTicket, Message: "ticket 4 overdue"}}
	_, body := Summarize(mixed, DetailMinimal)
	if body != "3 new notifications in Argus (1 critical, 1 high, 1 other)" {
		t.Fatalf("mixed = %q", body)
	}
}

func TestSummarizeDetailedAddsBoundedLines(t *testing.T) {
	var notes []domain.Notification
	for i := int64(1); i <= 7; i++ {
		notes = append(notes, finding(i, domain.SeverityLow))
	}
	_, body := Summarize(notes, DetailDetailed)
	if !strings.Contains(body, "- low finding on api.example.test") {
		t.Fatalf("detail missing: %q", body)
	}
	if strings.Count(body, "\n- ") != maxDetailLines || !strings.HasSuffix(body, "...and 2 more") {
		t.Fatalf("detail not bounded: %q", body)
	}
}

func TestRunOnceBaselineThenNewOnly(t *testing.T) {
	cursor := filepath.Join(t.TempDir(), "state", "cursor.json")
	src := &fakeSource{notes: []domain.Notification{finding(1, domain.SeverityCritical), finding(2, domain.SeverityHigh)}}
	snd := &fakeSender{}
	sent, err := RunOnce(context.Background(), src, snd, cursor, DetailMinimal)
	if err != nil || sent != 0 || len(snd.bodies) != 0 {
		t.Fatalf("baseline: sent=%d err=%v bodies=%v", sent, err, snd.bodies)
	}
	src.notes = append(src.notes, finding(3, domain.SeverityCritical))
	sent, err = RunOnce(context.Background(), src, snd, cursor, DetailMinimal)
	if err != nil || sent != 1 {
		t.Fatalf("new: sent=%d err=%v", sent, err)
	}
	if snd.bodies[0] != "1 new critical finding in Argus" || snd.titles[0] != Title {
		t.Fatalf("message = %q / %q", snd.titles[0], snd.bodies[0])
	}
	sent, err = RunOnce(context.Background(), src, snd, cursor, DetailMinimal)
	if err != nil || sent != 0 || len(snd.bodies) != 1 {
		t.Fatalf("repeat should not resend: sent=%d err=%v", sent, err)
	}
}

func TestRunOnceKeepsCursorWhenSendFails(t *testing.T) {
	cursor := filepath.Join(t.TempDir(), "cursor.json")
	src := &fakeSource{notes: []domain.Notification{finding(1, domain.SeverityHigh)}}
	if _, err := RunOnce(context.Background(), src, &fakeSender{}, cursor, DetailMinimal); err != nil {
		t.Fatal(err)
	}
	src.notes = append(src.notes, finding(2, domain.SeverityHigh))
	failing := &fakeSender{err: errors.New("no display")}
	if _, err := RunOnce(context.Background(), src, failing, cursor, DetailMinimal); err == nil {
		t.Fatal("expected send error")
	}
	retry := &fakeSender{}
	sent, err := RunOnce(context.Background(), src, retry, cursor, DetailMinimal)
	if err != nil || sent != 1 {
		t.Fatalf("retry: sent=%d err=%v", sent, err)
	}
}

func TestRunOnceSourceError(t *testing.T) {
	src := &fakeSource{err: errors.New("database locked")}
	if _, err := RunOnce(context.Background(), src, &fakeSender{}, filepath.Join(t.TempDir(), "c.json"), DetailMinimal); err == nil {
		t.Fatal("expected error")
	}
}

func TestParseDetailAndInterval(t *testing.T) {
	if _, err := ParseDetail("everything"); err == nil {
		t.Fatal("bad detail accepted")
	}
	if err := ValidateInterval(time.Second); err == nil {
		t.Fatal("interval below minimum accepted")
	}
	if err := ValidateInterval(DefaultInterval); err != nil {
		t.Fatal(err)
	}
}

func TestRunStopsOnCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	src := &fakeSource{}
	err := Run(ctx, src, &fakeSender{}, filepath.Join(t.TempDir(), "c.json"), DetailMinimal, time.Minute, func(string, ...any) {})
	if err != nil {
		t.Fatal(err)
	}
}
