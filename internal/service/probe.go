package service

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/NexzaDev/argus-asm/internal/domain"
	"github.com/NexzaDev/argus-asm/internal/probe"
)

const (
	maxProbePorts       = 64
	maxProbeConcurrency = 16
	defaultProbeTimeout = 2 * time.Second
	minProbeTimeout     = 100 * time.Millisecond
	maxProbeTimeout     = 10 * time.Second
	probeSource         = "tcp-connect"
)

type ProbeRequest struct {
	Asset       string
	Ports       []int
	Execute     bool
	Timeout     time.Duration
	Concurrency int
}

type ProbeResult struct {
	Asset    string `json:"asset"`
	Planned  []int  `json:"planned_ports"`
	Executed bool   `json:"executed"`
	Open     []int  `json:"open_ports"`
	Closed   int    `json:"closed_ports"`
}

func (s *Service) Probe(ctx context.Context, actor domain.Actor, req ProbeRequest, dial probe.Dialer) (ProbeResult, error) {
	if err := actor.Require(domain.PermOperate); err != nil {
		return ProbeResult{}, err
	}
	ports, err := normalizePorts(req.Ports)
	if err != nil {
		return ProbeResult{}, err
	}
	timeout := req.Timeout
	if timeout == 0 {
		timeout = defaultProbeTimeout
	}
	if timeout < minProbeTimeout || timeout > maxProbeTimeout {
		return ProbeResult{}, fmt.Errorf("%w: timeout must be between %s and %s", domain.ErrValidation, minProbeTimeout, maxProbeTimeout)
	}
	concurrency := req.Concurrency
	if concurrency == 0 {
		concurrency = 4
	}
	if concurrency < 1 || concurrency > maxProbeConcurrency {
		return ProbeResult{}, fmt.Errorf("%w: concurrency must be between 1 and %d", domain.ErrValidation, maxProbeConcurrency)
	}
	asset, err := s.resolveAsset(ctx, req.Asset)
	if err != nil {
		return ProbeResult{}, err
	}
	if !asset.ProbeAllowed {
		return ProbeResult{}, fmt.Errorf("%w: active probing of %s is not allowed; an administrator must run 'asset probe allow'", domain.ErrForbidden, asset.Value)
	}
	scopes, err := s.store.ListScopes(ctx)
	if err != nil {
		return ProbeResult{}, err
	}
	if _, err := authorizingScope(scopes, asset.Kind, asset.Value); err != nil {
		return ProbeResult{}, err
	}
	result := ProbeResult{Asset: asset.Value, Planned: ports, Executed: req.Execute, Open: []int{}}
	if !req.Execute {
		return result, nil
	}
	if dial == nil {
		return ProbeResult{}, fmt.Errorf("%w: no probe dialer configured", domain.ErrValidation)
	}
	open, closed, err := runProbe(ctx, dial, asset.Value, ports, timeout, concurrency)
	if err != nil {
		return ProbeResult{}, err
	}
	result.Open = open
	result.Closed = closed
	now := s.now()
	for _, port := range open {
		if _, err := s.store.UpsertService(ctx, domain.ServiceRecord{
			AssetID:    asset.ID,
			AssetValue: asset.Value,
			Port:       port,
			Protocol:   "tcp",
			Source:     probeSource,
		}, now); err != nil {
			return result, err
		}
	}
	detail := fmt.Sprintf("asset=%s planned=%d open=%d", asset.Value, len(ports), len(open))
	if err := s.audit(ctx, actor, "probe.tcp", detail); err != nil {
		return result, err
	}
	return result, nil
}

func normalizePorts(raw []int) ([]int, error) {
	if len(raw) == 0 {
		return nil, fmt.Errorf("%w: at least one port is required", domain.ErrValidation)
	}
	seen := map[int]bool{}
	var ports []int
	for _, p := range raw {
		if p < 1 || p > 65535 {
			return nil, fmt.Errorf("%w: port %d is outside 1-65535", domain.ErrValidation, p)
		}
		if !seen[p] {
			seen[p] = true
			ports = append(ports, p)
		}
	}
	if len(ports) > maxProbePorts {
		return nil, fmt.Errorf("%w: at most %d distinct ports may be probed per run", domain.ErrValidation, maxProbePorts)
	}
	sort.Ints(ports)
	return ports, nil
}

func runProbe(ctx context.Context, dial probe.Dialer, host string, ports []int, timeout time.Duration, concurrency int) ([]int, int, error) {
	jobs := make(chan int)
	var mu sync.Mutex
	var open []int
	closed := 0
	var firstErr error
	var wg sync.WaitGroup
	for w := 0; w < concurrency; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for port := range jobs {
				isOpen, err := dial(ctx, host, port, timeout)
				mu.Lock()
				switch {
				case err != nil && firstErr == nil:
					firstErr = err
				case isOpen:
					open = append(open, port)
				default:
					closed++
				}
				mu.Unlock()
			}
		}()
	}
	for _, port := range ports {
		if ctx.Err() != nil {
			break
		}
		jobs <- port
	}
	close(jobs)
	wg.Wait()
	if firstErr != nil {
		return nil, 0, firstErr
	}
	if err := ctx.Err(); err != nil {
		return nil, 0, err
	}
	sort.Ints(open)
	return open, closed, nil
}
