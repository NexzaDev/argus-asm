package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/NexzaDev/argus-asm/internal/domain"
)

const minSearchLength = 2

type SearchResult struct {
	Assets   []domain.Asset         `json:"assets"`
	Findings []FindingView          `json:"findings"`
	Services []domain.ServiceRecord `json:"services"`
}

func (s *Service) Search(ctx context.Context, actor domain.Actor, term string) (SearchResult, error) {
	if err := actor.Require(domain.PermRead); err != nil {
		return SearchResult{}, err
	}
	needle := strings.ToLower(strings.TrimSpace(term))
	if len(needle) < minSearchLength {
		return SearchResult{}, fmt.Errorf("%w: search term must be at least %d characters", domain.ErrValidation, minSearchLength)
	}
	result := SearchResult{Assets: []domain.Asset{}, Findings: []FindingView{}, Services: []domain.ServiceRecord{}}

	assets, err := s.store.ListAssets(ctx)
	if err != nil {
		return SearchResult{}, err
	}
	for _, a := range assets {
		if strings.Contains(strings.ToLower(a.Value), needle) || strings.Contains(strings.ToLower(a.OwnerName), needle) {
			result.Assets = append(result.Assets, a)
		}
	}
	views, err := s.findingViews(ctx, FindingFilter{IncludeResolved: true})
	if err != nil {
		return SearchResult{}, err
	}
	for _, v := range views {
		if strings.Contains(strings.ToLower(v.Title), needle) || strings.Contains(strings.ToLower(v.AssetValue), needle) {
			result.Findings = append(result.Findings, v)
		}
	}
	services, err := s.store.ListServices(ctx, 0)
	if err != nil {
		return SearchResult{}, err
	}
	for _, svc := range services {
		if strings.Contains(strings.ToLower(svc.Name), needle) || strings.Contains(strings.ToLower(svc.AssetValue), needle) {
			result.Services = append(result.Services, svc)
		}
	}
	return result, nil
}
