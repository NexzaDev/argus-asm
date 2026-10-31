package domain

import (
	"net/netip"
	"strings"
	"time"
)

type ScopeKind string

const (
	ScopeKindDomain ScopeKind = "domain"
	ScopeKindCIDR   ScopeKind = "cidr"
)

const (
	minIPv4ScopeBits = 16
	minIPv6ScopeBits = 32
)

type Scope struct {
	ID        int64     `json:"id"`
	Kind      ScopeKind `json:"kind"`
	Value     string    `json:"value"`
	CreatedAt time.Time `json:"created_at"`
}

func NewScope(raw string) (Scope, error) {
	value := strings.TrimSpace(raw)
	if strings.Contains(value, "/") {
		prefix, err := netip.ParsePrefix(value)
		if err != nil {
			return Scope{}, validationErrorf("invalid CIDR %q: %v", raw, err)
		}
		prefix = prefix.Masked()
		minBits := minIPv6ScopeBits
		if prefix.Addr().Is4() {
			minBits = minIPv4ScopeBits
		}
		if prefix.Bits() < minBits {
			return Scope{}, validationErrorf("CIDR %q is too broad (minimum prefix length is /%d)", raw, minBits)
		}
		return Scope{Kind: ScopeKindCIDR, Value: prefix.String()}, nil
	}
	name, err := NormalizeDomain(value)
	if err != nil {
		return Scope{}, err
	}
	return Scope{Kind: ScopeKindDomain, Value: name}, nil
}

func (s Scope) Authorizes(kind AssetKind, value string) bool {
	switch s.Kind {
	case ScopeKindDomain:
		return kind == AssetKindDomain && (value == s.Value || strings.HasSuffix(value, "."+s.Value))
	case ScopeKindCIDR:
		if kind != AssetKindIP {
			return false
		}
		prefix, err := netip.ParsePrefix(s.Value)
		if err != nil {
			return false
		}
		addr, err := netip.ParseAddr(value)
		if err != nil {
			return false
		}
		return prefix.Contains(addr)
	default:
		return false
	}
}
