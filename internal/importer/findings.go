package importer

import (
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/NexzaDev/argus-asm/internal/domain"
)

type Limits struct {
	MaxBytes       int64
	MaxRows        int
	MaxFieldLength int
}

var DefaultLimits = Limits{MaxBytes: 5 << 20, MaxRows: 10000, MaxFieldLength: 512}

const utf8BOM = "\ufeff"

var requiredColumns = []string{"asset", "title", "severity", "source"}

type Row struct {
	Line     int
	Asset    string
	Title    string
	Source   string
	Severity domain.Severity
}

type RowError struct {
	Line    int    `json:"line"`
	Message string `json:"message"`
}

type Result struct {
	Rows      []Row
	RowErrors []RowError
}

func ParseFindings(r io.Reader, limits Limits) (Result, error) {
	budget := &io.LimitedReader{R: r, N: limits.MaxBytes + 1}
	reader := csv.NewReader(budget)
	reader.FieldsPerRecord = -1
	reader.TrimLeadingSpace = true

	header, err := reader.Read()
	if err != nil {
		return Result{}, readError(budget, limits, err)
	}
	header[0] = strings.TrimPrefix(header[0], utf8BOM)
	columns, err := columnIndex(header)
	if err != nil {
		return Result{}, err
	}

	var result Result
	for {
		record, err := reader.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return Result{}, readError(budget, limits, err)
		}
		line, _ := reader.FieldPos(0)
		if len(result.Rows)+len(result.RowErrors) >= limits.MaxRows {
			return Result{}, fmt.Errorf("%w: findings file has more than %d rows", domain.ErrValidation, limits.MaxRows)
		}
		row, err := parseRecord(record, columns, len(header), line, limits.MaxFieldLength)
		if err != nil {
			result.RowErrors = append(result.RowErrors, RowError{Line: line, Message: err.Error()})
			continue
		}
		result.Rows = append(result.Rows, row)
	}
	if budget.N <= 0 {
		return Result{}, tooLargeError(limits)
	}
	return result, nil
}

func tooLargeError(limits Limits) error {
	return fmt.Errorf("%w: findings file exceeds %d bytes", domain.ErrValidation, limits.MaxBytes)
}

func readError(budget *io.LimitedReader, limits Limits, err error) error {
	if budget.N <= 0 {
		return tooLargeError(limits)
	}
	return fmt.Errorf("%w: %v", domain.ErrValidation, err)
}

func columnIndex(header []string) (map[string]int, error) {
	columns := make(map[string]int, len(header))
	for i, name := range header {
		key := strings.ToLower(strings.TrimSpace(name))
		if _, exists := columns[key]; exists {
			return nil, fmt.Errorf("%w: duplicate column %q in header", domain.ErrValidation, key)
		}
		columns[key] = i
	}
	for _, required := range requiredColumns {
		if _, ok := columns[required]; !ok {
			return nil, fmt.Errorf("%w: missing required column %q (header must include %s)",
				domain.ErrValidation, required, strings.Join(requiredColumns, ", "))
		}
	}
	return columns, nil
}

func parseRecord(record []string, columns map[string]int, headerLength, line, maxFieldLength int) (Row, error) {
	if len(record) != headerLength {
		return Row{}, fmt.Errorf("expected %d fields, found %d", headerLength, len(record))
	}
	return buildRow(line, maxFieldLength, rawFields{
		Asset:    record[columns["asset"]],
		Title:    record[columns["title"]],
		Source:   record[columns["source"]],
		Severity: record[columns["severity"]],
	})
}

type rawFields struct {
	Asset    string
	Title    string
	Source   string
	Severity string
}

func buildRow(line, maxFieldLength int, fields rawFields) (Row, error) {
	asset := strings.TrimSpace(fields.Asset)
	title := strings.TrimSpace(fields.Title)
	source := strings.TrimSpace(fields.Source)
	if asset == "" {
		return Row{}, errors.New("asset is empty")
	}
	if title == "" {
		return Row{}, errors.New("title is empty")
	}
	if source == "" {
		return Row{}, errors.New("source is empty")
	}
	for name, value := range map[string]string{"asset": asset, "title": title, "source": source} {
		if len(value) > maxFieldLength {
			return Row{}, fmt.Errorf("%s exceeds %d characters", name, maxFieldLength)
		}
	}
	severity, err := domain.ParseSeverity(fields.Severity)
	if err != nil {
		return Row{}, err
	}
	return Row{Line: line, Asset: asset, Title: title, Source: source, Severity: severity}, nil
}
