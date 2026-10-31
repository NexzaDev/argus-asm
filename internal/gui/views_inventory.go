//go:build !nogui

package gui

import (
	"fmt"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"github.com/NexzaDev/argus-asm/internal/domain"
	"github.com/NexzaDev/argus-asm/internal/service"
)

func stamp(t time.Time) string {
	if t.IsZero() {
		return "-"
	}
	return t.Local().Format("2006-01-02 15:04")
}

func emptyNote(what string) string {
	return "No " + what + " stored yet. Use the terminal commands in the Usage guide to add some."
}

func buildScope(s *session) fyne.CanvasObject {
	data := tableData{Columns: []string{"Kind", "Value", "Added"}, Widths: []float32{120, 320, 180}, Empty: emptyNote("scopes")}
	for _, scope := range s.snap.Scopes {
		data.Rows = append(data.Rows, []string{string(scope.Kind), scope.Value, stamp(scope.CreatedAt)})
		data.Details = append(data.Details, fmt.Sprintf("Scope %s\nKind: %s\nAdded: %s\n\nAssets are accepted only when a scope authorizes them.", scope.Value, scope.Kind, stamp(scope.CreatedAt)))
	}
	return newTableView(s, "Scope", data).object()
}

func buildAssets(s *session) fyne.CanvasObject {
	data := tableData{Columns: []string{"Kind", "Asset", "Owner", "Verified by", "Probe allowed", "Added"}, Widths: []float32{100, 260, 140, 140, 120, 150}, Empty: emptyNote("assets")}
	for _, asset := range s.snap.Assets {
		owner := asset.OwnerName
		if owner == "" {
			owner = "-"
		}
		verified := asset.VerifiedBy
		if verified == "" {
			verified = "-"
		}
		data.Rows = append(data.Rows, []string{string(asset.Kind), asset.Value, owner, verified, yesNo(asset.ProbeAllowed), stamp(asset.CreatedAt)})
		data.Details = append(data.Details, fmt.Sprintf("Asset %s\nKind: %s\nOwner: %s\nVerified by: %s\nProbe allowed: %s\nAdded: %s\nScope status: %s",
			asset.Value, asset.Kind, owner, verified, yesNo(asset.ProbeAllowed), stamp(asset.CreatedAt), scopeStatus(s.snap.Scopes, asset)))
	}
	return newTableView(s, "Assets", data).object()
}

func yesNo(value bool) string {
	if value {
		return "yes"
	}
	return "no"
}

func scopeStatus(scopes []domain.Scope, asset domain.Asset) string {
	for _, scope := range scopes {
		if scope.Authorizes(asset.Kind, asset.Value) {
			return "authorized by scope " + scope.Value
		}
	}
	return "outside every registered scope"
}

var severityRank = map[domain.Severity]string{
	domain.SeverityCritical: "1", domain.SeverityHigh: "2", domain.SeverityMedium: "3", domain.SeverityLow: "4", domain.SeverityInfo: "5",
}

func buildFindings(s *session) fyne.CanvasObject {
	data := tableData{
		Columns:     []string{"Severity", "Risk score", "Status", "Asset", "Title", "Source", "First seen", "Last seen"},
		Widths:      []float32{120, 100, 110, 220, 260, 160, 150, 150},
		HasSeverity: true,
		SeverityCol: 0,
		Empty:       emptyNote("findings"),
	}
	for _, f := range s.snap.Findings {
		data.Rows = append(data.Rows, []string{
			string(f.Severity), fmt.Sprint(f.Risk.Score), string(f.Status), f.AssetValue, f.Title, f.Source, stamp(f.FirstSeen), stamp(f.LastSeen),
		})
		data.Keys = append(data.Keys, []string{severityRank[f.Severity], fmt.Sprintf("%03d", f.Risk.Score), string(f.Status), f.AssetValue, f.Title, f.Source, stamp(f.FirstSeen), stamp(f.LastSeen)})
		data.Details = append(data.Details, findingDetail(f))
	}
	return newTableView(s, "Findings", data).object()
}

func findingDetail(f service.FindingView) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s\nAsset: %s\nSeverity: %s\nStatus: %s\nSource: %s\nRisk score: %d\n", f.Title, f.AssetValue, f.Severity, f.Status, f.Source, f.Risk.Score)
	fmt.Fprintf(&b, "First seen: %s\nLast seen: %s\n", stamp(f.FirstSeen), stamp(f.LastSeen))
	if f.StatusReason != "" {
		fmt.Fprintf(&b, "Status reason: %s\n", f.StatusReason)
	}
	b.WriteString("\nScore explanation:\n")
	for _, factor := range f.Risk.Factors {
		fmt.Fprintf(&b, "- %s\n", factor)
	}
	return b.String()
}

func buildImports(s *session) fyne.CanvasObject {
	data := tableData{Columns: []string{"Batch", "Imported", "Format", "Source", "Accepted", "Created", "Updated"}, Widths: []float32{90, 160, 110, 180, 110, 110, 110}, Empty: emptyNote("imports")}
	for _, batch := range s.snap.Batches {
		data.Rows = append(data.Rows, []string{fmt.Sprint(batch.ID), stamp(batch.CreatedAt), batch.Format, batch.Source, fmt.Sprint(batch.Accepted), fmt.Sprint(batch.Created), fmt.Sprint(batch.Updated)})
		data.Details = append(data.Details, fmt.Sprintf("Batch %d\nImported: %s\nFormat: %s\nSource: %s\nAccepted rows: %d\nCreated findings: %d\nUpdated findings: %d",
			batch.ID, stamp(batch.CreatedAt), batch.Format, batch.Source, batch.Accepted, batch.Created, batch.Updated))
	}
	return newTableView(s, "Imports", data).object()
}

func buildChanges(s *session) fyne.CanvasObject {
	sources := distinctSources(s.snap.Batches)
	if len(sources) == 0 {
		return wrapped("No imports yet. Changes compare two imports from the same source.")
	}
	summary := wrapped("")
	show := func(source string) {
		var report service.ChangeReport
		err := s.withService(func(svc *service.Service, actor domain.Actor) error {
			var err error
			report, err = svc.Changes(s.ctx, actor, source)
			return err
		})
		if err != nil {
			summary.SetText(err.Error())
			return
		}
		data := changeRows(report)
		summary.SetText(fmt.Sprintf("Source %s: %d new, %d missing, %d severity changed. Latest import %s, previous %s.",
			source, len(report.New), len(report.Missing), len(report.SeverityChanged), stamp(report.Latest.CreatedAt), previousText(report)))
		table := newTableView(s, "Changes", data)
		s.changesBody.Objects = []fyne.CanvasObject{table.object()}
		s.changesBody.Refresh()
	}
	s.changesBody = container.NewStack()
	picker := widget.NewSelect(sources, nil)
	picker.SetSelected(sources[0])
	picker.OnChanged = show
	show(sources[0])
	return container.NewBorder(container.NewVBox(widget.NewLabel("Source"), picker, summary), nil, nil, nil, s.changesBody)
}

func previousText(report service.ChangeReport) string {
	if report.Previous.ID == 0 {
		return "none (only one import from this source)"
	}
	return stamp(report.Previous.CreatedAt)
}

func changeRows(report service.ChangeReport) tableData {
	data := tableData{Columns: []string{"Change", "Asset", "Title", "Severity", "Detail"}, Widths: []float32{130, 220, 260, 200, 260}, Empty: "No changes between these two imports."}
	for _, obs := range report.New {
		data.Rows = append(data.Rows, []string{"new", obs.AssetValue, obs.Title, string(obs.Severity), "present in the latest import only"})
		data.Details = append(data.Details, "")
	}
	for _, obs := range report.Missing {
		data.Rows = append(data.Rows, []string{"missing", obs.AssetValue, obs.Title, string(obs.Severity), "present in the previous import only"})
		data.Details = append(data.Details, "")
	}
	for _, change := range report.SeverityChanged {
		data.Rows = append(data.Rows, []string{"severity changed", change.AssetValue, change.Title, string(change.From) + " to " + string(change.To), "severity moved between imports"})
		data.Details = append(data.Details, "")
	}
	return data
}

func distinctSources(batches []domain.ImportBatch) []string {
	seen := map[string]bool{}
	var out []string
	for _, batch := range batches {
		if !seen[batch.Source] {
			seen[batch.Source] = true
			out = append(out, batch.Source)
		}
	}
	return out
}

func buildAccess(s *session) fyne.CanvasObject {
	data := tableData{Columns: []string{"User", "Role", "Added"}, Widths: []float32{240, 160, 180}, Empty: "No users yet. The local bootstrap actor is used until users are added in the terminal."}
	for _, user := range s.snap.Users {
		data.Rows = append(data.Rows, []string{user.Name, string(user.Role), stamp(user.CreatedAt)})
		data.Details = append(data.Details, fmt.Sprintf("User %s\nRole: %s\nAdded: %s\n\nRoles are changed in a terminal with 'argus-asm user role <name> --role <role>'.", user.Name, user.Role, stamp(user.CreatedAt)))
	}
	return newTableView(s, "Access", data).object()
}

func buildAudit(s *session) fyne.CanvasObject {
	verdict := "Audit chain verified: " + fmt.Sprint(s.snap.Verify.Chained) + " chained entries, " + fmt.Sprint(s.snap.Verify.Legacy) + " legacy entries without a chain."
	if !s.snap.Verify.OK {
		verdict = "Audit chain problem: " + s.snap.Verify.Problem
	}
	banner := wrapped(verdict)
	data := tableData{Columns: []string{"Time", "Actor", "Action", "Detail", "Hash (first 12)"}, Widths: []float32{160, 120, 170, 320, 150}, Empty: "No audit entries yet."}
	for _, entry := range s.snap.Audit {
		hash := entry.Hash
		if len(hash) > 12 {
			hash = hash[:12]
		}
		data.Rows = append(data.Rows, []string{stamp(entry.At), entry.Actor, entry.Action, entry.Detail, hash})
		data.Details = append(data.Details, fmt.Sprintf("Entry %d\nTime: %s\nActor: %s\nAction: %s\nDetail: %s\nPrevious hash: %s\nHash: %s",
			entry.ID, stamp(entry.At), entry.Actor, entry.Action, entry.Detail, entry.PrevHash, entry.Hash))
	}
	return container.NewBorder(banner, nil, nil, nil, newTableView(s, "Audit", data).object())
}
