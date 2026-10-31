package importer

import (
	"strings"
	"testing"
)

func FuzzParseNmapXML(f *testing.F) {
	f.Add(`<nmaprun><host><status state="up"/><address addr="192.0.2.10" addrtype="ipv4"/><ports><port protocol="tcp" portid="22"><state state="open"/></port></ports></host></nmaprun>`)
	f.Add(`<!DOCTYPE nmaprun><nmaprun></nmaprun>`)
	f.Add(`<!DOCTYPE nmaprun [<!ENTITY x "y">]><nmaprun>&x;</nmaprun>`)
	f.Fuzz(func(t *testing.T, input string) {
		_, _ = ParseNmapXML(strings.NewReader(input), DefaultLimits)
	})
}

func FuzzParseFindingsJSON(f *testing.F) {
	f.Add(`[{"asset":"www.example.test","title":"x","severity":"low","source":"s"}]`)
	f.Add(`[{"asset":"192.0.2.1","title":"x","severity":"critical","source":"s","extra":true}]`)
	f.Add(`[] []`)
	f.Fuzz(func(t *testing.T, input string) {
		_, _ = ParseFindingsJSON(strings.NewReader(input), DefaultLimits)
	})
}

func FuzzParseFindingsSARIF(f *testing.F) {
	f.Add(`{"version":"2.1.0","runs":[{"tool":{"driver":{"name":"s"}},"results":[{"ruleId":"R","level":"error","locations":[{"physicalLocation":{"artifactLocation":{"uri":"192.0.2.1"}}}],"properties":{"security-severity":"7.5"}}]}]}`)
	f.Add(`{"version":"2.1.0","runs":[{"results":[{"message":{"text":"x"}}]}]}`)
	f.Add(`{"version":"2.0.0","runs":[]}`)
	f.Fuzz(func(t *testing.T, input string) {
		_, _ = ParseFindingsSARIF(strings.NewReader(input), DefaultLimits)
	})
}
