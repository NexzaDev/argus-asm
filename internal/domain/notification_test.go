package domain

import "testing"

func TestNotificationSeverityFromFindingMessage(t *testing.T) {
	cases := []struct {
		name string
		n    Notification
		want Severity
		ok   bool
	}{
		{"critical finding", Notification{Kind: NotificationFinding, Message: "critical finding on api.example.test: x (src)"}, SeverityCritical, true},
		{"info finding", Notification{Kind: NotificationFinding, Message: "info finding on 192.0.2.10: y (src)"}, SeverityInfo, true},
		{"ticket has no severity", Notification{Kind: NotificationTicket, Message: "critical finding on a: b"}, "", false},
		{"unknown prefix", Notification{Kind: NotificationFinding, Message: "severe finding on a"}, "", false},
	}
	for _, tc := range cases {
		got, ok := NotificationSeverity(tc.n)
		if got != tc.want || ok != tc.ok {
			t.Fatalf("%s: got (%q,%v), want (%q,%v)", tc.name, got, ok, tc.want, tc.ok)
		}
	}
}
