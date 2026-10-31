package sqlite

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
)

const maxSnapshotBytes = 1 << 30

func (s *Store) Snapshot(ctx context.Context) ([]byte, error) {
	suffix := make([]byte, 8)
	if _, err := rand.Read(suffix); err != nil {
		return nil, fmt.Errorf("random snapshot name: %w", err)
	}
	tmp := filepath.Join(filepath.Dir(s.path), ".argus-asm-snapshot-"+hex.EncodeToString(suffix)+".tmp")
	file, err := os.OpenFile(tmp, os.O_CREATE|os.O_EXCL|os.O_WRONLY, dataFileMode)
	if err != nil {
		return nil, fmt.Errorf("create snapshot file: %w", err)
	}
	if err := file.Close(); err != nil {
		_ = os.Remove(tmp)
		return nil, fmt.Errorf("close snapshot file: %w", err)
	}
	defer func() { _ = os.Remove(tmp) }()
	if _, err := s.db.ExecContext(ctx, `VACUUM INTO ?`, tmp); err != nil {
		return nil, fmt.Errorf("write snapshot: %w", err)
	}
	info, err := os.Stat(tmp)
	if err != nil {
		return nil, fmt.Errorf("inspect snapshot: %w", err)
	}
	if info.Size() > maxSnapshotBytes {
		return nil, fmt.Errorf("database is larger than %d bytes", maxSnapshotBytes)
	}
	data, err := os.ReadFile(tmp)
	if err != nil {
		return nil, fmt.Errorf("read snapshot: %w", err)
	}
	return data, nil
}
