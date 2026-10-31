package importer

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
)

const worstRows = 10000

func worstTitle() string {
	return strings.Repeat("A", 440)
}

func worstCSV() []byte {
	var b strings.Builder
	b.WriteString("asset,title,severity,source\n")
	title := worstTitle()
	for i := 0; i < worstRows; i++ {
		fmt.Fprintf(&b, "api.example.test,%s,critical,bench\n", title)
	}
	return []byte(b.String())
}

func worstJSON() []byte {
	var b strings.Builder
	b.WriteString("[")
	title := worstTitle()
	for i := 0; i < worstRows; i++ {
		if i > 0 {
			b.WriteString(",")
		}
		fmt.Fprintf(&b, `{"asset":"api.example.test","title":"%s","severity":"critical","source":"bench"}`, title)
	}
	b.WriteString("]")
	return []byte(b.String())
}

func worstSARIF() []byte {
	var b strings.Builder
	b.WriteString(`{"version":"2.1.0","runs":[{"tool":{"driver":{"name":"bench"}},"results":[`)
	text := strings.Repeat("m", 300)
	for i := 0; i < worstRows; i++ {
		if i > 0 {
			b.WriteString(",")
		}
		fmt.Fprintf(&b, `{"ruleId":"BENCH-%05d","level":"error","message":{"text":"%s"},"locations":[{"physicalLocation":{"artifactLocation":{"uri":"api.example.test"}}}],"properties":{"security-severity":"9.1"}}`, i, text)
	}
	b.WriteString("]}]}")
	return []byte(b.String())
}

func worstNmap() []byte {
	var b strings.Builder
	b.WriteString(`<?xml version="1.0"?><nmaprun scanner="nmap" version="7.94">`)
	for i := 0; i < worstRows; i++ {
		fmt.Fprintf(&b, `<host><status state="up"/><address addr="198.51.100.%d" addrtype="ipv4"/><hostnames/><ports><port protocol="tcp" portid="22"><state state="open"/><service name="ssh"/></port></ports></host>`, i%250)
	}
	b.WriteString(`</nmaprun>`)
	return []byte(b.String())
}

func TestWorstCaseInputsAreAccepted(t *testing.T) {
	csvData := worstCSV()
	if int64(len(csvData)) > DefaultLimits.MaxBytes {
		t.Fatalf("CSV worst case is %d bytes, above the limit", len(csvData))
	}
	res, err := ParseFindings(bytes.NewReader(csvData), DefaultLimits)
	if err != nil || len(res.Rows) != worstRows {
		t.Fatalf("CSV worst case: rows=%d err=%v", len(res.Rows), err)
	}
	jsonData := worstJSON()
	if int64(len(jsonData)) > DefaultLimits.MaxBytes {
		t.Fatalf("JSON worst case is %d bytes, above the limit", len(jsonData))
	}
	jres, err := ParseFindingsJSON(bytes.NewReader(jsonData), DefaultLimits)
	if err != nil || len(jres.Rows) != worstRows {
		t.Fatalf("JSON worst case: rows=%d err=%v", len(jres.Rows), err)
	}
	sarifData := worstSARIF()
	if int64(len(sarifData)) > DefaultLimits.MaxBytes {
		t.Fatalf("SARIF worst case is %d bytes, above the limit", len(sarifData))
	}
	sres, err := ParseFindingsSARIF(bytes.NewReader(sarifData), DefaultLimits)
	if err != nil || len(sres.Rows) != worstRows {
		t.Fatalf("SARIF worst case: rows=%d err=%v", len(sres.Rows), err)
	}
	nmapData := worstNmap()
	if int64(len(nmapData)) > DefaultLimits.MaxBytes {
		t.Fatalf("Nmap worst case is %d bytes, above the limit", len(nmapData))
	}
	nres, err := ParseNmapXML(bytes.NewReader(nmapData), DefaultLimits)
	if err != nil || len(nres.Services) != worstRows {
		t.Fatalf("Nmap worst case: services=%d err=%v", len(nres.Services), err)
	}
}

func BenchmarkParseFindingsCSVWorstCase(b *testing.B) {
	data := worstCSV()
	b.SetBytes(int64(len(data)))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := ParseFindings(bytes.NewReader(data), DefaultLimits); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkParseFindingsJSONWorstCase(b *testing.B) {
	data := worstJSON()
	b.SetBytes(int64(len(data)))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := ParseFindingsJSON(bytes.NewReader(data), DefaultLimits); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkParseFindingsSARIFWorstCase(b *testing.B) {
	data := worstSARIF()
	b.SetBytes(int64(len(data)))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := ParseFindingsSARIF(bytes.NewReader(data), DefaultLimits); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkParseNmapXMLWorstCase(b *testing.B) {
	data := worstNmap()
	b.SetBytes(int64(len(data)))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := ParseNmapXML(bytes.NewReader(data), DefaultLimits); err != nil {
			b.Fatal(err)
		}
	}
}
