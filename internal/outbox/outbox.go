package outbox

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"time"

	"github.com/NexzaDev/argus-asm/internal/domain"
)

const fileMode fs.FileMode = 0o600

type record struct {
	ID        int64     `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	Kind      string    `json:"kind"`
	Message   string    `json:"message"`
}

func Check(path string) error {
	if path == "" {
		return fmt.Errorf("%w: outbox path is required", domain.ErrValidation)
	}
	info, err := os.Lstat(path)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return nil
	case err != nil:
		return fmt.Errorf("inspect outbox: %w", err)
	case info.Mode()&fs.ModeSymlink != 0:
		return fmt.Errorf("%w: outbox %s is a symbolic link; use a regular file", domain.ErrValidation, path)
	case !info.Mode().IsRegular():
		return fmt.Errorf("%w: outbox %s is not a regular file", domain.ErrValidation, path)
	}
	return nil
}

func Open(path string) (*os.File, error) {
	if err := Check(path); err != nil {
		return nil, err
	}
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_APPEND|os.O_CREATE, fileMode)
	if err != nil {
		return nil, fmt.Errorf("open outbox: %w", err)
	}
	return file, nil
}

func Write(w io.Writer, notes []domain.Notification) error {
	encoder := json.NewEncoder(w)
	for _, n := range notes {
		if err := encoder.Encode(record{ID: n.ID, CreatedAt: n.CreatedAt, Kind: string(n.Kind), Message: n.Message}); err != nil {
			return fmt.Errorf("write outbox record %d: %w", n.ID, err)
		}
	}
	return nil
}
