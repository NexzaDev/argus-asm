package outbox

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/NexzaDev/argus-asm/internal/domain"
)

func TestOpenCreatesFileWithOwnerOnlyMode(t *testing.T) {
	path := filepath.Join(t.TempDir(), "outbox.jsonl")
	file, err := Open(path)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	file.Close()
	if runtime.GOOS != "windows" {
		info, err := os.Stat(path)
		if err != nil || info.Mode().Perm() != 0o600 {
			t.Fatalf("mode = %v, %v; want 0600", info.Mode().Perm(), err)
		}
	}
}

func TestOpenAppendsToExistingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "outbox.jsonl")
	if err := os.WriteFile(path, []byte("existing\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	file, err := Open(path)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	if _, err := file.WriteString("next\n"); err != nil {
		t.Fatal(err)
	}
	file.Close()
	data, _ := os.ReadFile(path)
	if string(data) != "existing\nnext\n" {
		t.Fatalf("contents = %q", data)
	}
}

func TestOpenRefusesEmptyPathDirectoriesAndSymlinks(t *testing.T) {
	dir := t.TempDir()
	if _, err := Open(""); err == nil {
		t.Fatal("empty path accepted")
	}
	if _, err := Open(dir); err == nil {
		t.Fatal("directory accepted")
	}
	if runtime.GOOS == "windows" {
		return
	}
	target := filepath.Join(dir, "target.txt")
	if err := os.WriteFile(target, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "link.jsonl")
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if _, err := Open(link); err == nil || !strings.Contains(err.Error(), "symbolic link") {
		t.Fatalf("symlink error = %v", err)
	}
}

func TestWriteEmitsOneJSONObjectPerLine(t *testing.T) {
	at := time.Date(2026, 10, 29, 9, 0, 0, 0, time.UTC)
	notes := []domain.Notification{
		{ID: 7, CreatedAt: at, Kind: domain.NotificationTicket, Message: "line one\nline two"},
		{ID: 8, CreatedAt: at, Kind: domain.NotificationCertificate, Message: "certificate expires"},
	}
	var buf bytes.Buffer
	if err := Write(&buf, notes); err != nil {
		t.Fatalf("Write() error = %v", err)
	}
	lines := strings.Split(strings.TrimSuffix(buf.String(), "\n"), "\n")
	if len(lines) != 2 {
		t.Fatalf("lines = %d, want 2: %q", len(lines), buf.String())
	}
	var first map[string]any
	if err := json.Unmarshal([]byte(lines[0]), &first); err != nil {
		t.Fatalf("line 1 is not JSON: %v", err)
	}
	if first["id"] != float64(7) || first["message"] != "line one\nline two" || first["kind"] != "ticket" {
		t.Fatalf("record = %v", first)
	}
}
