package cli

import (
	"encoding/json"
	"io"
	"text/tabwriter"
	"time"
)

func writeJSON(w io.Writer, value any) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}

func newTable(w io.Writer) *tabwriter.Writer {
	return tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
}

func formatTimestamp(t time.Time) string {
	return t.UTC().Format(time.RFC3339)
}

func formatOptionalTimestamp(t *time.Time) string {
	if t == nil {
		return "-"
	}
	return formatTimestamp(*t)
}
