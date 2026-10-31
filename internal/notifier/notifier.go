package notifier

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/NexzaDev/argus-asm/internal/domain"
)

const (
	Title           = "Argus ASM"
	DefaultInterval = 5 * time.Minute
	MinInterval     = time.Minute
	MaxInterval     = 24 * time.Hour
	maxDetailLines  = 5
	maxDetailChars  = 120
	cursorDirMode   = 0o700
	cursorFileMode  = 0o600
)

type Detail string

const (
	DetailMinimal  Detail = "minimal"
	DetailDetailed Detail = "detailed"
)

func ParseDetail(value string) (Detail, error) {
	switch Detail(value) {
	case DetailMinimal, DetailDetailed:
		return Detail(value), nil
	default:
		return "", fmt.Errorf("detail must be %q or %q, got %q", DetailMinimal, DetailDetailed, value)
	}
}

func ValidateInterval(interval time.Duration) error {
	if interval < MinInterval || interval > MaxInterval {
		return fmt.Errorf("interval must be between %s and %s, got %s", MinInterval, MaxInterval, interval)
	}
	return nil
}

type Source interface {
	Notifications(ctx context.Context) ([]domain.Notification, error)
}

type Sender interface {
	Send(ctx context.Context, title, body string) error
}

var severityOrder = []domain.Severity{domain.SeverityCritical, domain.SeverityHigh, domain.SeverityMedium, domain.SeverityLow, domain.SeverityInfo}

func Summarize(notes []domain.Notification, detail Detail) (string, string) {
	counts := map[domain.Severity]int{}
	other := 0
	for _, n := range notes {
		if sev, ok := domain.NotificationSeverity(n); ok {
			counts[sev]++
		} else {
			other++
		}
	}
	var body string
	if other == 0 && len(counts) == 1 {
		for _, sev := range severityOrder {
			if c := counts[sev]; c > 0 {
				body = fmt.Sprintf("%d new %s %s in Argus", c, sev, plural(c, "finding", "findings"))
			}
		}
	} else {
		body = fmt.Sprintf("%d new %s in Argus", len(notes), plural(len(notes), "notification", "notifications"))
		var parts []string
		for _, sev := range severityOrder {
			if c := counts[sev]; c > 0 {
				parts = append(parts, fmt.Sprintf("%d %s", c, sev))
			}
		}
		if other > 0 {
			parts = append(parts, fmt.Sprintf("%d other", other))
		}
		if len(parts) > 0 {
			body += " (" + strings.Join(parts, ", ") + ")"
		}
	}
	if detail == DetailDetailed {
		for i, n := range notes {
			if i == maxDetailLines {
				body += fmt.Sprintf("\n...and %d more", len(notes)-maxDetailLines)
				break
			}
			body += "\n- " + truncate(n.Message, maxDetailChars)
		}
	}
	return Title, body
}

func plural(n int, one, many string) string {
	if n == 1 {
		return one
	}
	return many
}

func truncate(text string, limit int) string {
	runes := []rune(text)
	if len(runes) <= limit {
		return text
	}
	return string(runes[:limit-1]) + "…"
}

type cursorFile struct {
	LastID int64 `json:"last_id"`
}

func readCursor(path string) (int64, bool, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, fmt.Errorf("read notifier cursor: %w", err)
	}
	var cur cursorFile
	if err := json.Unmarshal(data, &cur); err != nil {
		return 0, false, fmt.Errorf("parse notifier cursor %s: %w", path, err)
	}
	return cur.LastID, true, nil
}

func WriteCursor(path string, lastID int64) error {
	if err := os.MkdirAll(filepath.Dir(path), cursorDirMode); err != nil {
		return fmt.Errorf("create notifier state directory: %w", err)
	}
	data, err := json.Marshal(cursorFile{LastID: lastID})
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, append(data, '\n'), cursorFileMode); err != nil {
		return fmt.Errorf("write notifier cursor: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("replace notifier cursor: %w", err)
	}
	return nil
}

func BaselineID(notes []domain.Notification) int64 {
	var max int64
	for _, n := range notes {
		if n.ID > max {
			max = n.ID
		}
	}
	return max
}

func RunOnce(ctx context.Context, src Source, snd Sender, cursorPath string, detail Detail) (int, error) {
	all, err := src.Notifications(ctx)
	if err != nil {
		return 0, fmt.Errorf("read notifications: %w", err)
	}
	last, found, err := readCursor(cursorPath)
	if err != nil {
		return 0, err
	}
	if !found {
		return 0, WriteCursor(cursorPath, BaselineID(all))
	}
	var fresh []domain.Notification
	for _, n := range all {
		if n.ID > last {
			fresh = append(fresh, n)
		}
	}
	if len(fresh) == 0 {
		return 0, nil
	}
	title, body := Summarize(fresh, detail)
	if err := snd.Send(ctx, title, body); err != nil {
		return 0, fmt.Errorf("send notification: %w", err)
	}
	if err := WriteCursor(cursorPath, BaselineID(fresh)); err != nil {
		return len(fresh), err
	}
	return len(fresh), nil
}

func Run(ctx context.Context, src Source, snd Sender, cursorPath string, detail Detail, interval time.Duration, logf func(format string, args ...any)) error {
	if err := ValidateInterval(interval); err != nil {
		return err
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		sent, err := RunOnce(ctx, src, snd, cursorPath, detail)
		switch {
		case err != nil:
			logf("notifier: %v", err)
		case sent > 0:
			logf("notifier: sent notification for %d new item(s)", sent)
		}
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
		}
	}
}
