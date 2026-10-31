package domain

import (
	"strings"
	"time"
)

type FindingStatus string

const (
	FindingOpen     FindingStatus = "open"
	FindingAccepted FindingStatus = "accepted"
	FindingResolved FindingStatus = "resolved"
)

const maxStatusReasonLength = 500

func ParseFindingStatus(raw string) (FindingStatus, error) {
	switch status := FindingStatus(strings.ToLower(strings.TrimSpace(raw))); status {
	case FindingOpen, FindingAccepted, FindingResolved:
		return status, nil
	}
	return "", validationErrorf("unknown finding status %q (expected open, accepted, or resolved)", raw)
}

func ValidateFindingTransition(from, to FindingStatus, reason string) (string, error) {
	if from == to {
		return "", validationErrorf("finding is already %s", to)
	}
	trimmed := strings.TrimSpace(reason)
	if len(trimmed) > maxStatusReasonLength {
		return "", validationErrorf("status reason exceeds %d characters", maxStatusReasonLength)
	}
	if to == FindingAccepted && trimmed == "" {
		return "", validationErrorf("accepting a risk requires --reason explaining the decision")
	}
	return trimmed, nil
}

type TicketStatus string

const (
	TicketOpen       TicketStatus = "open"
	TicketInProgress TicketStatus = "in_progress"
	TicketDone       TicketStatus = "done"
)

func ParseTicketStatus(raw string) (TicketStatus, error) {
	switch status := TicketStatus(strings.ToLower(strings.TrimSpace(raw))); status {
	case TicketOpen, TicketInProgress, TicketDone:
		return status, nil
	}
	return "", validationErrorf("unknown ticket status %q (expected open, in_progress, or done)", raw)
}

func ValidateTicketTransition(from, to TicketStatus) error {
	if from == to {
		return validationErrorf("ticket is already %s", to)
	}
	return nil
}

func ParseDueDate(raw string) (time.Time, error) {
	due, err := time.Parse("2006-01-02", strings.TrimSpace(raw))
	if err != nil {
		return time.Time{}, validationErrorf("due date %q must use YYYY-MM-DD", raw)
	}
	return due, nil
}

func ValidateOwner(raw string) (string, error) {
	owner := strings.TrimSpace(raw)
	if owner == "" || len(owner) > 64 {
		return "", validationErrorf("owner must be 1-64 characters")
	}
	return owner, nil
}
