package domain

import (
	"net/netip"
	"strings"
)

const (
	maxDomainLength = 253
	maxLabelLength  = 63
)

func NormalizeDomain(raw string) (string, error) {
	name := strings.ToLower(strings.TrimSuffix(strings.TrimSpace(raw), "."))
	if name == "" {
		return "", validationErrorf("domain name is empty")
	}
	if len(name) > maxDomainLength {
		return "", validationErrorf("domain %q exceeds %d characters", raw, maxDomainLength)
	}
	if _, err := netip.ParseAddr(name); err == nil {
		return "", validationErrorf("%q is an IP address; use an IP scope or asset instead", raw)
	}
	labels := strings.Split(name, ".")
	if len(labels) < 2 {
		return "", validationErrorf("domain %q must contain at least two labels", raw)
	}
	for _, label := range labels {
		if !isValidLabel(label) {
			return "", validationErrorf("domain %q contains invalid label %q", raw, label)
		}
	}
	return name, nil
}

func isValidLabel(label string) bool {
	if len(label) == 0 || len(label) > maxLabelLength || label[0] == '-' || label[len(label)-1] == '-' {
		return false
	}
	for i := 0; i < len(label); i++ {
		c := label[i]
		if !(c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '-') {
			return false
		}
	}
	return true
}
