package importer

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/NexzaDev/argus-asm/internal/domain"
)

type jsonFinding struct {
	Asset    string `json:"asset"`
	Title    string `json:"title"`
	Severity string `json:"severity"`
	Source   string `json:"source"`
}

func ParseFindingsJSON(r io.Reader, limits Limits) (Result, error) {
	data, err := io.ReadAll(io.LimitReader(r, limits.MaxBytes+1))
	if err != nil {
		return Result{}, fmt.Errorf("%w: read findings JSON: %v", domain.ErrValidation, err)
	}
	if int64(len(data)) > limits.MaxBytes {
		return Result{}, tooLargeError(limits)
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	token, err := decoder.Token()
	if err != nil {
		return Result{}, fmt.Errorf("%w: findings JSON must be an array of objects: %v", domain.ErrValidation, err)
	}
	if delim, ok := token.(json.Delim); !ok || delim != '[' {
		return Result{}, fmt.Errorf("%w: findings JSON must be an array of objects", domain.ErrValidation)
	}

	var result Result
	record := 0
	for decoder.More() {
		record++
		if record > limits.MaxRows {
			return Result{}, fmt.Errorf("%w: findings file has more than %d rows", domain.ErrValidation, limits.MaxRows)
		}
		var raw json.RawMessage
		if err := decoder.Decode(&raw); err != nil {
			return Result{}, fmt.Errorf("%w: record %d: %v", domain.ErrValidation, record, err)
		}
		row, err := parseJSONRecord(raw, record, limits.MaxFieldLength)
		if err != nil {
			result.RowErrors = append(result.RowErrors, RowError{Line: record, Message: err.Error()})
			continue
		}
		result.Rows = append(result.Rows, row)
	}
	if _, err := decoder.Token(); err != nil {
		return Result{}, fmt.Errorf("%w: findings JSON is truncated: %v", domain.ErrValidation, err)
	}
	if _, err := decoder.Token(); !errors.Is(err, io.EOF) {
		return Result{}, fmt.Errorf("%w: unexpected data after the findings array", domain.ErrValidation)
	}
	return result, nil
}

func parseJSONRecord(raw json.RawMessage, record, maxFieldLength int) (Row, error) {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	var item jsonFinding
	if err := decoder.Decode(&item); err != nil {
		return Row{}, fmt.Errorf("invalid object: %v", err)
	}
	return buildRow(record, maxFieldLength, rawFields{
		Asset:    item.Asset,
		Title:    item.Title,
		Source:   item.Source,
		Severity: item.Severity,
	})
}
