package importer

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"strings"

	"github.com/NexzaDev/argus-asm/internal/domain"
)

const maxNmapNameLength = 128

type NmapService struct {
	Host     int
	Targets  []string
	Port     int
	Protocol string
	Name     string
}

type NmapResult struct {
	Services  []NmapService
	RowErrors []RowError
}

type nmapRun struct {
	XMLName xml.Name   `xml:"nmaprun"`
	Hosts   []nmapHost `xml:"host"`
}

type nmapHost struct {
	Status    nmapState     `xml:"status"`
	Addresses []nmapAddress `xml:"address"`
	Hostnames struct {
		Names []nmapHostname `xml:"hostname"`
	} `xml:"hostnames"`
	Ports struct {
		Ports []nmapPort `xml:"port"`
	} `xml:"ports"`
}

type nmapState struct {
	State string `xml:"state,attr"`
}

type nmapAddress struct {
	Addr     string `xml:"addr,attr"`
	AddrType string `xml:"addrtype,attr"`
}

type nmapHostname struct {
	Name string `xml:"name,attr"`
}

type nmapPort struct {
	Protocol string    `xml:"protocol,attr"`
	PortID   int       `xml:"portid,attr"`
	State    nmapState `xml:"state"`
	Service  struct {
		Name string `xml:"name,attr"`
	} `xml:"service"`
}

func ParseNmapXML(r io.Reader, limits Limits) (NmapResult, error) {
	data, err := io.ReadAll(io.LimitReader(r, limits.MaxBytes+1))
	if err != nil {
		return NmapResult{}, fmt.Errorf("%w: read nmap XML: %v", domain.ErrValidation, err)
	}
	if int64(len(data)) > limits.MaxBytes {
		return NmapResult{}, tooLargeError(limits)
	}
	if hasInternalDTD(data) {
		return NmapResult{}, fmt.Errorf("%w: nmap XML must not contain an internal DTD subset or entity declarations", domain.ErrValidation)
	}
	var run nmapRun
	if err := xml.Unmarshal(data, &run); err != nil {
		return NmapResult{}, fmt.Errorf("%w: parse nmap XML: %v", domain.ErrValidation, err)
	}
	if run.XMLName.Local != "nmaprun" {
		return NmapResult{}, fmt.Errorf("%w: root element must be nmaprun", domain.ErrValidation)
	}
	if len(run.Hosts) > limits.MaxRows {
		return NmapResult{}, fmt.Errorf("%w: nmap XML has more than %d hosts", domain.ErrValidation, limits.MaxRows)
	}

	var result NmapResult
	for i, host := range run.Hosts {
		hostNumber := i + 1
		if host.Status.State != "up" {
			continue
		}
		targets := hostTargets(host)
		if len(targets) == 0 {
			result.RowErrors = append(result.RowErrors, RowError{Line: hostNumber, Message: "host has no usable address or hostname"})
			continue
		}
		for _, port := range host.Ports.Ports {
			if port.State.State != "open" {
				continue
			}
			if len(result.Services) >= limits.MaxRows {
				return NmapResult{}, fmt.Errorf("%w: nmap XML has more than %d open services", domain.ErrValidation, limits.MaxRows)
			}
			name := strings.TrimSpace(port.Service.Name)
			if len(name) > maxNmapNameLength {
				result.RowErrors = append(result.RowErrors, RowError{Line: hostNumber,
					Message: fmt.Sprintf("service name on port %d exceeds %d characters", port.PortID, maxNmapNameLength)})
				continue
			}
			result.Services = append(result.Services, NmapService{
				Host:     hostNumber,
				Targets:  targets,
				Port:     port.PortID,
				Protocol: strings.ToLower(port.Protocol),
				Name:     name,
			})
		}
	}
	return result, nil
}

func hasInternalDTD(data []byte) bool {
	if bytes.Contains(data, []byte("<!ENTITY")) {
		return true
	}
	start := bytes.Index(data, []byte("<!DOCTYPE"))
	if start < 0 {
		return false
	}
	end := bytes.IndexByte(data[start:], '>')
	if end < 0 {
		return true
	}
	return bytes.IndexByte(data[start:start+end], '[') >= 0
}

func hostTargets(host nmapHost) []string {
	var targets []string
	for _, address := range host.Addresses {
		if address.AddrType == "ipv4" || address.AddrType == "ipv6" {
			targets = append(targets, address.Addr)
		}
	}
	for _, name := range host.Hostnames.Names {
		if trimmed := strings.TrimSpace(name.Name); trimmed != "" {
			targets = append(targets, trimmed)
		}
	}
	return targets
}
