package domain

import (
	"errors"
	"testing"
	"time"
)

func TestNormalizeDomain(t *testing.T) {
	valid := map[string]string{
		"Example.TEST":        "example.test",
		" api.example.test. ": "api.example.test",
		"a-b.c1.example":      "a-b.c1.example",
	}
	for input, want := range valid {
		got, err := NormalizeDomain(input)
		if err != nil || got != want {
			t.Fatalf("NormalizeDomain(%q) = %q, %v; want %q", input, got, err, want)
		}
	}
	invalid := []string{"", "localhost", "-bad.example", "bad-.example", "under_score.example",
		"double..dot.example", "192.0.2.1", "*.example.test", "spa ce.example",
		"xn--" + repeat('a', 60) + ".example", repeat('a', 64) + ".example"}
	for _, input := range invalid {
		if _, err := NormalizeDomain(input); !errors.Is(err, ErrValidation) {
			t.Fatalf("NormalizeDomain(%q) error = %v; want ErrValidation", input, err)
		}
	}
}

func TestNormalizeDomainRejectsOverlongName(t *testing.T) {
	long := repeat('a', 63) + "." + repeat('b', 63) + "." + repeat('c', 63) + "." + repeat('d', 63) + ".test"
	if _, err := NormalizeDomain(long); !errors.Is(err, ErrValidation) {
		t.Fatalf("expected validation error for %d-character name, got %v", len(long), err)
	}
}

func TestNewScope(t *testing.T) {
	scope, err := NewScope("example.test")
	if err != nil || scope.Kind != ScopeKindDomain || scope.Value != "example.test" {
		t.Fatalf("domain scope = %+v, %v", scope, err)
	}
	scope, err = NewScope("192.0.2.77/24")
	if err != nil || scope.Kind != ScopeKindCIDR || scope.Value != "192.0.2.0/24" {
		t.Fatalf("cidr scope = %+v, %v (host bits must be masked)", scope, err)
	}
	for _, input := range []string{"0.0.0.0/0", "10.0.0.0/8", "::/0", "2001:db8::/16", "192.0.2.0/33", "not-a-cidr/24"} {
		if _, err := NewScope(input); !errors.Is(err, ErrValidation) {
			t.Fatalf("NewScope(%q) error = %v; want ErrValidation", input, err)
		}
	}
}

func TestScopeAuthorizes(t *testing.T) {
	domainScope, _ := NewScope("example.test")
	cidrScope, _ := NewScope("192.0.2.0/24")
	cases := []struct {
		name  string
		scope Scope
		kind  AssetKind
		value string
		want  bool
	}{
		{"apex", domainScope, AssetKindDomain, "example.test", true},
		{"subdomain", domainScope, AssetKindDomain, "deep.api.example.test", true},
		{"suffix lookalike", domainScope, AssetKindDomain, "evilexample.test", false},
		{"parent", domainScope, AssetKindDomain, "test", false},
		{"ip under domain scope", domainScope, AssetKindIP, "192.0.2.1", false},
		{"ip in range", cidrScope, AssetKindIP, "192.0.2.200", true},
		{"ip outside range", cidrScope, AssetKindIP, "192.0.3.1", false},
		{"domain under cidr scope", cidrScope, AssetKindDomain, "example.test", false},
		{"unknown scope kind", Scope{Kind: "bogus"}, AssetKindDomain, "example.test", false},
	}
	for _, tc := range cases {
		if got := tc.scope.Authorizes(tc.kind, tc.value); got != tc.want {
			t.Fatalf("%s: Authorizes(%s, %s) = %v; want %v", tc.name, tc.kind, tc.value, got, tc.want)
		}
	}
}

func TestParseAsset(t *testing.T) {
	kind, value, err := ParseAsset("::ffff:192.0.2.5")
	if err != nil || kind != AssetKindIP || value != "192.0.2.5" {
		t.Fatalf("mapped IPv4 = %s %s %v", kind, value, err)
	}
	kind, value, err = ParseAsset("2001:DB8::1")
	if err != nil || kind != AssetKindIP || value != "2001:db8::1" {
		t.Fatalf("IPv6 = %s %s %v", kind, value, err)
	}
	if _, _, err := ParseAsset("fe80::1%eth0"); !errors.Is(err, ErrValidation) {
		t.Fatalf("zoned IP error = %v", err)
	}
	if _, _, err := ParseAsset("192.0.2.0/24"); !errors.Is(err, ErrValidation) {
		t.Fatalf("CIDR as asset error = %v", err)
	}
	if _, _, err := ParseAsset("bad host"); !errors.Is(err, ErrValidation) {
		t.Fatalf("invalid host error = %v", err)
	}
}

func TestParseSeverityAndWeights(t *testing.T) {
	severity, err := ParseSeverity(" HIGH ")
	if err != nil || severity != SeverityHigh {
		t.Fatalf("ParseSeverity = %q, %v", severity, err)
	}
	if _, err := ParseSeverity("urgent"); !errors.Is(err, ErrValidation) {
		t.Fatalf("unknown severity error = %v", err)
	}
	order := []Severity{SeverityCritical, SeverityHigh, SeverityMedium, SeverityLow, SeverityInfo}
	for i := 1; i < len(order); i++ {
		if order[i-1].Weight() <= order[i].Weight() {
			t.Fatalf("weight(%s) must exceed weight(%s)", order[i-1], order[i])
		}
	}
	if Severity("bogus").Weight() != 0 {
		t.Fatal("unknown severity must weigh zero")
	}
}

func TestDedupeKeyIsStableAndCaseInsensitive(t *testing.T) {
	a := DedupeKey(AssetKindDomain, "api.example.test", "Title", "Scanner")
	b := DedupeKey(AssetKindDomain, "api.example.test", " title ", "scanner")
	c := DedupeKey(AssetKindIP, "api.example.test", "Title", "Scanner")
	if a != b {
		t.Fatal("dedupe key must ignore case and surrounding whitespace in title and source")
	}
	if a == c {
		t.Fatal("dedupe key must distinguish asset kinds")
	}
}

func TestAssessRiskExplainsScore(t *testing.T) {
	assessment := AssessRisk(SeverityHigh, FindingOpen, "", ExposureContext{})
	if assessment.Score != 70 || len(assessment.Factors) != 1 {
		t.Fatalf("AssessRisk(high) = %+v", assessment)
	}
}

func TestAssessRiskAddsExposureAndAging(t *testing.T) {
	assessment := AssessRisk(SeverityMedium, FindingOpen, "", ExposureContext{OpenServices: 3, DaysOpen: 45})
	if assessment.Score != 40+6+5 {
		t.Fatalf("score = %d, want 51; factors %v", assessment.Score, assessment.Factors)
	}
	if len(assessment.Factors) != 3 {
		t.Fatalf("factors = %v, want severity, exposure, and aging", assessment.Factors)
	}
}

func TestAssessRiskCapsServiceExposure(t *testing.T) {
	assessment := AssessRisk(SeverityLow, FindingOpen, "", ExposureContext{OpenServices: 50})
	if assessment.Score != 15+10 {
		t.Fatalf("score = %d, want 25 (services capped at 5)", assessment.Score)
	}
}

func TestAssessRiskZeroesAcceptedAndResolved(t *testing.T) {
	accepted := AssessRisk(SeverityCritical, FindingAccepted, "business need", ExposureContext{OpenServices: 5})
	if accepted.Score != 0 {
		t.Fatalf("accepted score = %d, want 0", accepted.Score)
	}
	resolved := AssessRisk(SeverityCritical, FindingResolved, "", ExposureContext{})
	if resolved.Score != 0 {
		t.Fatalf("resolved score = %d, want 0", resolved.Score)
	}
}

func TestValidateFindingTransition(t *testing.T) {
	if _, err := ValidateFindingTransition(FindingOpen, FindingAccepted, "  "); !errors.Is(err, ErrValidation) {
		t.Fatalf("accepting without reason: err = %v, want validation error", err)
	}
	reason, err := ValidateFindingTransition(FindingOpen, FindingAccepted, "  compensating control  ")
	if err != nil || reason != "compensating control" {
		t.Fatalf("reason = %q, err = %v", reason, err)
	}
	if _, err := ValidateFindingTransition(FindingOpen, FindingOpen, ""); !errors.Is(err, ErrValidation) {
		t.Fatalf("same status: err = %v, want validation error", err)
	}
}

func TestRolesGrantExpectedPermissions(t *testing.T) {
	cases := []struct {
		role Role
		perm Permission
		want bool
	}{
		{RoleViewer, PermRead, true},
		{RoleViewer, PermOperate, false},
		{RoleAnalyst, PermOperate, true},
		{RoleAnalyst, PermAdminister, false},
		{RoleAdmin, PermAdminister, true},
	}
	for _, c := range cases {
		if got := c.role.Grants(c.perm); got != c.want {
			t.Errorf("%s.Grants(%s) = %t, want %t", c.role, c.perm, got, c.want)
		}
	}
}

func TestActorRequireReturnsForbidden(t *testing.T) {
	err := Actor{Name: "vera", Role: RoleViewer}.Require(PermOperate)
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("err = %v, want ErrForbidden", err)
	}
}

func TestAuditChainDetectsTampering(t *testing.T) {
	at := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	first := AuditEntry{ID: 1, At: at, Actor: "local", Action: "scope.add", Detail: "domain example.test"}
	first.Hash = AuditHash("", first.At, first.Actor, first.Action, first.Detail)
	second := AuditEntry{ID: 2, At: at.Add(time.Second), Actor: "local", Action: "asset.add", Detail: "domain www.example.test", PrevHash: first.Hash}
	second.Hash = AuditHash(second.PrevHash, second.At, second.Actor, second.Action, second.Detail)

	if result := VerifyAuditChain([]AuditEntry{first, second}); !result.OK || result.Chained != 2 {
		t.Fatalf("intact chain: %+v", result)
	}
	tampered := second
	tampered.Detail = "domain evil.test"
	if result := VerifyAuditChain([]AuditEntry{first, tampered}); result.OK {
		t.Fatal("modified detail must fail verification")
	}
	if result := VerifyAuditChain([]AuditEntry{first, {ID: 3, At: at, Actor: "x", Action: "y", PrevHash: "wrong", Hash: "z"}}); result.OK {
		t.Fatal("broken link must fail verification")
	}
}

func TestParseAssetRules(t *testing.T) {
	kind, value, err := ParseAsset("::ffff:192.0.2.5")
	if err != nil || kind != AssetKindIP || value != "192.0.2.5" {
		t.Fatalf("mapped IPv4 = %s %s %v", kind, value, err)
	}
	if _, _, err := ParseAsset("192.0.2.0/24"); !errors.Is(err, ErrValidation) {
		t.Fatalf("CIDR as asset: err = %v, want validation error", err)
	}
	if _, _, err := ParseAsset("fe80::1%eth0"); !errors.Is(err, ErrValidation) {
		t.Fatalf("zoned IPv6: err = %v, want validation error", err)
	}
}

func TestParseRoleAndUserName(t *testing.T) {
	if _, err := ParseRole("superuser"); !errors.Is(err, ErrValidation) {
		t.Fatalf("unknown role: err = %v", err)
	}
	if _, err := ValidateUserName("bad name!"); !errors.Is(err, ErrValidation) {
		t.Fatalf("bad user name: err = %v", err)
	}
	if name, err := ValidateUserName(" alice.ops "); err != nil || name != "alice.ops" {
		t.Fatalf("valid user name = %q, %v", name, err)
	}
}

func repeat(char byte, count int) string {
	out := make([]byte, count)
	for i := range out {
		out[i] = char
	}
	return string(out)
}
