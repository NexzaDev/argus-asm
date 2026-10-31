package importer

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"testing"
	"time"

	"github.com/NexzaDev/argus-asm/internal/domain"
)

func TestParseFindingsJSONAcceptsValidArray(t *testing.T) {
	input := `[{"asset":"www.example.test","title":"Expired TLS","severity":"high","source":"scanner-a"},
	{"asset":"192.0.2.10","title":"SSH exposed","severity":"medium","source":"scanner-a"}]`
	result, err := ParseFindingsJSON(strings.NewReader(input), DefaultLimits)
	if err != nil {
		t.Fatalf("ParseFindingsJSON() error = %v", err)
	}
	if len(result.Rows) != 2 || len(result.RowErrors) != 0 {
		t.Fatalf("rows = %d, row errors = %v", len(result.Rows), result.RowErrors)
	}
	if result.Rows[0].Severity != domain.SeverityHigh || result.Rows[1].Line != 2 {
		t.Fatalf("unexpected rows: %+v", result.Rows)
	}
}

func TestParseFindingsJSONReportsRowErrors(t *testing.T) {
	input := `[{"asset":"www.example.test","title":"x","severity":"nope","source":"s"},
	{"asset":"www.example.test","title":"x","severity":"low","source":"s","extra":1}]`
	result, err := ParseFindingsJSON(strings.NewReader(input), DefaultLimits)
	if err != nil {
		t.Fatalf("ParseFindingsJSON() error = %v", err)
	}
	if len(result.RowErrors) != 2 || result.RowErrors[0].Line != 1 || result.RowErrors[1].Line != 2 {
		t.Fatalf("row errors = %+v", result.RowErrors)
	}
}

func TestParseFindingsJSONRejectsMalformedDocuments(t *testing.T) {
	for name, input := range map[string]string{
		"object instead of array": `{"asset":"a"}`,
		"truncated":               `[{"asset":"www.example.test"`,
		"trailing data":           `[] []`,
	} {
		if _, err := ParseFindingsJSON(strings.NewReader(input), DefaultLimits); !errors.Is(err, domain.ErrValidation) {
			t.Errorf("%s: err = %v, want validation error", name, err)
		}
	}
}

func TestParseFindingsJSONEnforcesRowLimit(t *testing.T) {
	limits := Limits{MaxBytes: 1 << 20, MaxRows: 1, MaxFieldLength: 512}
	input := `[{"asset":"www.example.test","title":"a","severity":"low","source":"s"},
	{"asset":"www.example.test","title":"b","severity":"low","source":"s"}]`
	if _, err := ParseFindingsJSON(strings.NewReader(input), limits); !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("row limit: err = %v, want validation error", err)
	}
}

func TestParseNmapXMLExtractsOpenPorts(t *testing.T) {
	input := `<?xml version="1.0"?>
<nmaprun>
  <host>
    <status state="up"/>
    <address addr="192.0.2.10" addrtype="ipv4"/>
    <hostnames><hostname name="www.example.test"/></hostnames>
    <ports>
      <port protocol="tcp" portid="443"><state state="open"/><service name="https"/></port>
      <port protocol="tcp" portid="23"><state state="closed"/><service name="telnet"/></port>
    </ports>
  </host>
  <host>
    <status state="down"/>
    <address addr="192.0.2.11" addrtype="ipv4"/>
  </host>
</nmaprun>`
	result, err := ParseNmapXML(strings.NewReader(input), DefaultLimits)
	if err != nil {
		t.Fatalf("ParseNmapXML() error = %v", err)
	}
	if len(result.Services) != 1 {
		t.Fatalf("services = %+v", result.Services)
	}
	svc := result.Services[0]
	if svc.Port != 443 || svc.Protocol != "tcp" || svc.Name != "https" || len(svc.Targets) != 2 {
		t.Fatalf("service = %+v", svc)
	}
}

func TestParseNmapXMLRejectsEntityDeclarations(t *testing.T) {
	input := `<?xml version="1.0"?><!DOCTYPE nmaprun [<!ENTITY x "y">]><nmaprun></nmaprun>`
	if _, err := ParseNmapXML(strings.NewReader(input), DefaultLimits); !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("entity declaration: err = %v, want validation error", err)
	}
}

func TestParseNmapXMLAcceptsPlainDoctype(t *testing.T) {
	input := `<?xml version="1.0"?>
<!DOCTYPE nmaprun>
<?xml-stylesheet href="file:///usr/share/nmap/nmap.xsl" type="text/xsl"?>
<nmaprun><host><status state="up"/><address addr="192.0.2.10" addrtype="ipv4"/>
<ports><port protocol="tcp" portid="22"><state state="open"/><service name="ssh"/></port></ports></host></nmaprun>`
	result, err := ParseNmapXML(strings.NewReader(input), DefaultLimits)
	if err != nil || len(result.Services) != 1 {
		t.Fatalf("plain DOCTYPE: services=%+v err=%v", result.Services, err)
	}
}

func TestParseNmapXMLRejectsInternalDTDSubset(t *testing.T) {
	input := `<?xml version="1.0"?><!DOCTYPE nmaprun [<!ENTITY x SYSTEM "file:///etc/passwd">]><nmaprun>&x;</nmaprun>`
	if _, err := ParseNmapXML(strings.NewReader(input), DefaultLimits); !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("internal subset: err = %v, want validation error", err)
	}
}

func TestParseNmapXMLRejectsWrongRoot(t *testing.T) {
	if _, err := ParseNmapXML(strings.NewReader(`<other/>`), DefaultLimits); !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("wrong root: err = %v, want validation error", err)
	}
}

func TestParseCertificatesPEMReadsCertificates(t *testing.T) {
	data := selfSignedPEM(t, "www.example.test", time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC))
	certs, err := ParseCertificatesPEM(data, DefaultLimits)
	if err != nil {
		t.Fatalf("ParseCertificatesPEM() error = %v", err)
	}
	if len(certs) != 1 || len(certs[0].Fingerprint) != 64 || !strings.Contains(certs[0].Subject, "www.example.test") {
		t.Fatalf("certs = %+v", certs)
	}
}

func TestParseCertificatesPEMRejectsNonCertificateInput(t *testing.T) {
	if _, err := ParseCertificatesPEM([]byte("not pem"), DefaultLimits); !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("plain text: err = %v, want validation error", err)
	}
}

func selfSignedPEM(t *testing.T, name string, notAfter time.Time) []byte {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: name},
		DNSNames:     []string{name},
		NotBefore:    time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		NotAfter:     notAfter,
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("create certificate: %v", err)
	}
	var buf bytes.Buffer
	if err := pem.Encode(&buf, &pem.Block{Type: "CERTIFICATE", Bytes: der}); err != nil {
		t.Fatalf("encode PEM: %v", err)
	}
	return buf.Bytes()
}

func TestParseNmapXMLEnforcesByteLimit(t *testing.T) {
	input := `<?xml version="1.0"?><nmaprun><host><status state="up"/><address addr="192.0.2.10" addrtype="ipv4"/><ports><port protocol="tcp" portid="443"><state state="open"/></port></ports></host></nmaprun>`
	size := int64(len(input))
	if _, err := ParseNmapXML(strings.NewReader(input), Limits{MaxBytes: size, MaxRows: 10, MaxFieldLength: 512}); err != nil {
		t.Fatalf("input at the exact byte limit: %v", err)
	}
	if _, err := ParseNmapXML(strings.NewReader(input), Limits{MaxBytes: size - 1, MaxRows: 10, MaxFieldLength: 512}); !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("input one byte over the limit: err = %v, want validation error", err)
	}
}

func TestParseCertificatesPEMEnforcesByteLimit(t *testing.T) {
	data := selfSignedPEM(t, "www.example.test", time.Now().Add(365*24*time.Hour))
	size := int64(len(data))
	if _, err := ParseCertificatesPEM(data, Limits{MaxBytes: size, MaxRows: 10, MaxFieldLength: 512}); err != nil {
		t.Fatalf("PEM at the exact byte limit: %v", err)
	}
	if _, err := ParseCertificatesPEM(data, Limits{MaxBytes: size - 1, MaxRows: 10, MaxFieldLength: 512}); !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("PEM one byte over the limit: err = %v, want validation error", err)
	}
}

func TestParseNmapXMLCapsOpenServicesAtMaxRows(t *testing.T) {
	var b strings.Builder
	b.WriteString(`<?xml version="1.0"?><nmaprun scanner="nmap"><host><status state="up"/><address addr="198.51.100.7" addrtype="ipv4"/><ports>`)
	for port := 1; port <= 5; port++ {
		fmt.Fprintf(&b, `<port protocol="tcp" portid="%d"><state state="open"/><service name="svc"/></port>`, port)
	}
	b.WriteString(`</ports></host></nmaprun>`)
	input := b.String()
	if _, err := ParseNmapXML(strings.NewReader(input), Limits{MaxBytes: 1 << 20, MaxRows: 5, MaxFieldLength: 512}); err != nil {
		t.Fatalf("5 open services at MaxRows 5: err = %v", err)
	}
	if _, err := ParseNmapXML(strings.NewReader(input), Limits{MaxBytes: 1 << 20, MaxRows: 4, MaxFieldLength: 512}); !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("5 open services at MaxRows 4: err = %v", err)
	}
}
