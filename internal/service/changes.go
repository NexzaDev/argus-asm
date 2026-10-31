package service

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/NexzaDev/argus-asm/internal/domain"
)

type SeverityChange struct {
	FindingID  int64           `json:"finding_id"`
	AssetValue string          `json:"asset"`
	Title      string          `json:"title"`
	From       domain.Severity `json:"from"`
	To         domain.Severity `json:"to"`
}

type ChangeReport struct {
	Source          string                    `json:"source"`
	Latest          domain.ImportBatch        `json:"latest"`
	Previous        domain.ImportBatch        `json:"previous"`
	New             []domain.BatchObservation `json:"new"`
	Missing         []domain.BatchObservation `json:"missing"`
	SeverityChanged []SeverityChange          `json:"severity_changed"`
}

func (s *Service) Changes(ctx context.Context, actor domain.Actor, source string) (ChangeReport, error) {
	if err := actor.Require(domain.PermRead); err != nil {
		return ChangeReport{}, err
	}
	name := strings.TrimSpace(source)
	if name == "" {
		return ChangeReport{}, fmt.Errorf("%w: --source is required to compare imports from one source", domain.ErrValidation)
	}
	batches, err := s.store.ListBatches(ctx, name)
	if err != nil {
		return ChangeReport{}, err
	}
	if len(batches) < 2 {
		return ChangeReport{}, fmt.Errorf("%w: source %q has %d import(s); at least two are needed to compare", domain.ErrValidation, name, len(batches))
	}
	latest, previous := batches[0], batches[1]
	latestObs, err := s.store.BatchObservations(ctx, latest.ID)
	if err != nil {
		return ChangeReport{}, err
	}
	previousObs, err := s.store.BatchObservations(ctx, previous.ID)
	if err != nil {
		return ChangeReport{}, err
	}
	report := ChangeReport{
		Source:          name,
		Latest:          latest,
		Previous:        previous,
		New:             []domain.BatchObservation{},
		Missing:         []domain.BatchObservation{},
		SeverityChanged: []SeverityChange{},
	}
	prevByID := make(map[int64]domain.BatchObservation, len(previousObs))
	for _, o := range previousObs {
		prevByID[o.FindingID] = o
	}
	latestByID := make(map[int64]domain.BatchObservation, len(latestObs))
	for _, o := range latestObs {
		latestByID[o.FindingID] = o
		prev, ok := prevByID[o.FindingID]
		if !ok {
			report.New = append(report.New, o)
			continue
		}
		if prev.Severity != o.Severity {
			report.SeverityChanged = append(report.SeverityChanged, SeverityChange{
				FindingID: o.FindingID, AssetValue: o.AssetValue, Title: o.Title, From: prev.Severity, To: o.Severity,
			})
		}
	}
	for _, o := range previousObs {
		if _, ok := latestByID[o.FindingID]; !ok {
			report.Missing = append(report.Missing, o)
		}
	}
	sort.Slice(report.SeverityChanged, func(i, j int) bool { return report.SeverityChanged[i].FindingID < report.SeverityChanged[j].FindingID })
	return report, nil
}
