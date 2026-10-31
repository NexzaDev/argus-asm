package importer

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"strconv"
	"strings"

	"github.com/NexzaDev/argus-asm/internal/domain"
)

const sarifVersion = "2.1.0"

type sarifLog struct {
	Version string     `json:"version"`
	Runs    []sarifRun `json:"runs"`
}

type sarifRun struct {
	Tool struct {
		Driver struct {
			Name string `json:"name"`
		} `json:"driver"`
	} `json:"tool"`
	Results []sarifResult `json:"results"`
}

type sarifResult struct {
	RuleID  string `json:"ruleId"`
	Level   string `json:"level"`
	Message struct {
		Text string `json:"text"`
	} `json:"message"`
	Locations []struct {
		PhysicalLocation struct {
			ArtifactLocation struct {
				URI string `json:"uri"`
			} `json:"artifactLocation"`
		} `json:"physicalLocation"`
	} `json:"locations"`
	Properties struct {
		SecuritySeverity json.RawMessage `json:"security-severity"`
	} `json:"properties"`
}

func ParseFindingsSARIF(r io.Reader, limits Limits) (Result, error) {
	data, err := io.ReadAll(io.LimitReader(r, limits.MaxBytes+1))
	if err != nil {
		return Result{}, fmt.Errorf("%w: read SARIF: %v", domain.ErrValidation, err)
	}
	if int64(len(data)) > limits.MaxBytes {
		return Result{}, tooLargeError(limits)
	}
	var log sarifLog
	if err := json.NewDecoder(bytes.NewReader(data)).Decode(&log); err != nil {
		return Result{}, fmt.Errorf("%w: SARIF document is not valid JSON: %v", domain.ErrValidation, err)
	}
	if log.Version != sarifVersion {
		return Result{}, fmt.Errorf("%w: SARIF version must be %q, found %q", domain.ErrValidation, sarifVersion, log.Version)
	}
	var result Result
	record := 0
	for _, run := range log.Runs {
		for _, item := range run.Results {
			record++
			if record > limits.MaxRows {
				return Result{}, fmt.Errorf("%w: SARIF file has more than %d results", domain.ErrValidation, limits.MaxRows)
			}
			row, err := sarifRow(record, run.Tool.Driver.Name, item, limits.MaxFieldLength)
			if err != nil {
				result.RowErrors = append(result.RowErrors, RowError{Line: record, Message: err.Error()})
				continue
			}
			result.Rows = append(result.Rows, row)
		}
	}
	return result, nil
}

func sarifRow(line int, driver string, item sarifResult, maxFieldLength int) (Row, error) {
	if len(item.Locations) == 0 {
		return Row{}, fmt.Errorf("result has no location; the asset is taken from the first location")
	}
	asset := sarifAsset(item.Locations[0].PhysicalLocation.ArtifactLocation.URI)
	title := strings.TrimSpace(item.RuleID)
	if title == "" {
		title = strings.TrimSpace(item.Message.Text)
	}
	severity, err := sarifSeverity(item.Properties.SecuritySeverity, item.Level)
	if err != nil {
		return Row{}, err
	}
	return buildRow(line, maxFieldLength, rawFields{
		Asset:    asset,
		Title:    title,
		Source:   driver,
		Severity: string(severity),
	})
}

func sarifAsset(uri string) string {
	value := strings.TrimSpace(uri)
	if parsed, err := url.Parse(value); err == nil && parsed.Host != "" {
		return parsed.Hostname()
	}
	return value
}

func sarifSeverity(score json.RawMessage, level string) (domain.Severity, error) {
	if value, ok := parseSecurityScore(score); ok {
		switch {
		case value >= 9.0:
			return domain.SeverityCritical, nil
		case value >= 7.0:
			return domain.SeverityHigh, nil
		case value >= 4.0:
			return domain.SeverityMedium, nil
		case value > 0:
			return domain.SeverityLow, nil
		default:
			return domain.SeverityInfo, nil
		}
	}
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "error":
		return domain.SeverityHigh, nil
	case "", "warning":
		return domain.SeverityMedium, nil
	case "note":
		return domain.SeverityLow, nil
	case "none":
		return domain.SeverityInfo, nil
	default:
		return "", fmt.Errorf("unknown SARIF level %q", level)
	}
}

func parseSecurityScore(raw json.RawMessage) (float64, bool) {
	text := strings.TrimSpace(string(raw))
	if text == "" || text == "null" {
		return 0, false
	}
	text = strings.Trim(text, `"`)
	value, err := strconv.ParseFloat(strings.TrimSpace(text), 64)
	if err != nil || value < 0 || value > 10 {
		return 0, false
	}
	return value, true
}
