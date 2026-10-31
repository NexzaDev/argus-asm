package domain

import "strings"

func NotificationSeverity(n Notification) (Severity, bool) {
	if n.Kind != NotificationFinding {
		return "", false
	}
	for _, sev := range []Severity{SeverityCritical, SeverityHigh, SeverityMedium, SeverityLow, SeverityInfo} {
		if strings.HasPrefix(n.Message, string(sev)+" finding on ") {
			return sev, true
		}
	}
	return "", false
}
