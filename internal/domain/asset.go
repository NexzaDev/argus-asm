package domain

import (
	"net/netip"
	"strings"
	"time"
)

type AssetKind string

const (
	AssetKindDomain AssetKind = "domain"
	AssetKindIP     AssetKind = "ip"
)

type Asset struct {
	ID           int64      `json:"id"`
	ScopeID      int64      `json:"scope_id"`
	Kind         AssetKind  `json:"kind"`
	Value        string     `json:"value"`
	OwnerName    string     `json:"owner,omitempty"`
	VerifiedBy   string     `json:"verified_by,omitempty"`
	VerifiedAt   *time.Time `json:"verified_at,omitempty"`
	ProbeAllowed bool       `json:"probe_allowed"`
	CreatedAt    time.Time  `json:"created_at"`
}

func ParseAsset(raw string) (AssetKind, string, error) {
	value := strings.TrimSpace(raw)
	if addr, err := netip.ParseAddr(value); err == nil {
		if addr.Zone() != "" {
			return "", "", validationErrorf("IP %q with a zone identifier is not supported", raw)
		}
		return AssetKindIP, addr.Unmap().String(), nil
	}
	if strings.Contains(value, "/") {
		return "", "", validationErrorf("%q is a range; register single addresses with 'asset add' and ranges with 'scope add'", raw)
	}
	name, err := NormalizeDomain(value)
	if err != nil {
		return "", "", err
	}
	return AssetKindDomain, name, nil
}
