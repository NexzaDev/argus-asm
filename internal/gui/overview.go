//go:build !nogui

package gui

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/NexzaDev/argus-asm/internal/domain"
	"github.com/NexzaDev/argus-asm/internal/service"
)

const staleAfterDays = 30

type riskTier struct {
	name  string
	count int
}

func tierOf(score int) string {
	switch {
	case score >= 90:
		return "Critical"
	case score >= 70:
		return "High"
	case score >= 40:
		return "Medium"
	}
	return "Low"
}

func heading(text string) fyne.CanvasObject {
	return widget.NewLabelWithStyle(text, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
}

type countBar struct {
	widget.BaseWidget
	value, max int
}

func newCountBar(value, max int) *countBar {
	b := &countBar{value: value, max: max}
	b.ExtendBaseWidget(b)
	return b
}

func (b *countBar) MinSize() fyne.Size {
	return fyne.NewSize(60, theme.TextSize())
}

func (b *countBar) CreateRenderer() fyne.WidgetRenderer {
	track := canvas.NewRectangle(theme.Color(theme.ColorNameInputBorder))
	fill := canvas.NewRectangle(theme.Color(theme.ColorNamePrimary))
	return &countBarRenderer{bar: b, track: track, fill: fill}
}

type countBarRenderer struct {
	bar   *countBar
	track *canvas.Rectangle
	fill  *canvas.Rectangle
}

func (r *countBarRenderer) Layout(size fyne.Size) {
	r.track.Resize(size)
	width := float32(0)
	if r.bar.max > 0 {
		width = size.Width * float32(r.bar.value) / float32(r.bar.max)
	}
	r.fill.Resize(fyne.NewSize(width, size.Height))
}

func (r *countBarRenderer) MinSize() fyne.Size {
	return r.bar.MinSize()
}

func (r *countBarRenderer) Refresh() {
	r.track.FillColor = theme.Color(theme.ColorNameInputBorder)
	r.fill.FillColor = theme.Color(theme.ColorNamePrimary)
	r.track.Refresh()
	r.fill.Refresh()
}

func (r *countBarRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{r.track, r.fill}
}

func (r *countBarRenderer) Destroy() {}

type flexLabel struct {
	widget.Label
}

func (l *flexLabel) MinSize() fyne.Size {
	size := l.Label.MinSize()
	size.Width = 0
	return size
}

func wrapped(text string) *flexLabel {
	label := &flexLabel{}
	label.ExtendBaseWidget(label)
	label.SetText(text)
	label.Wrapping = fyne.TextWrapWord
	return label
}

func buildOverview(s *session) fyne.CanvasObject {
	if s.loadErr != nil {
		return s.errorView(s.loadErr)
	}
	snap := s.snap
	open := openFindings(snap)
	counts := map[string]int{"Critical": 0, "High": 0, "Medium": 0, "Low": 0}
	for _, f := range open {
		counts[tierOf(f.Risk.Score)]++
	}
	tiers := []riskTier{{"Critical", counts["Critical"]}, {"High", counts["High"]}, {"Medium", counts["Medium"]}, {"Low", counts["Low"]}}
	maxCount := 1
	for _, tier := range tiers {
		if tier.count > maxCount {
			maxCount = tier.count
		}
	}

	objects := []fyne.CanvasObject{
		heading("Overview"),
		wrapped(fmt.Sprintf("Scopes %d    Assets %d    Open findings %d    Unread notifications %d", len(snap.Scopes), len(snap.Assets), len(open), unreadCount(snap.Notifications))),
		heading("Open findings by risk tier"),
		wrapped("Tiers use the risk score: Critical 90 and above, High 70 to 89, Medium 40 to 69, Low below 40. The bars and the text below show the same counts."),
	}
	for _, tier := range tiers {
		bar := newCountBar(tier.count, maxCount)
		row := container.NewBorder(nil, nil, widget.NewLabel(fmt.Sprintf("%s: %d", tier.name, tier.count)), nil, bar)
		objects = append(objects, row)
	}

	objects = append(objects, heading("Top open findings"))
	objects = append(objects, topFindings(open)...)

	objects = append(objects, heading("Assets outside scope or stale"))
	objects = append(objects, staleAssets(snap)...)

	objects = append(objects, heading("Last import"))
	objects = append(objects, lastImport(snap)...)

	objects = append(objects, heading("Changes since the previous import"))
	objects = append(objects, s.changesSummary(snap)...)

	return container.NewVScroll(container.NewPadded(container.NewVBox(objects...)))
}

func unreadCount(notifications []domain.Notification) int {
	count := 0
	for _, n := range notifications {
		if n.AcknowledgedAt == nil {
			count++
		}
	}
	return count
}

func topFindings(open []service.FindingView) []fyne.CanvasObject {
	if len(open) == 0 {
		return []fyne.CanvasObject{wrapped("No open findings. Import findings with 'argus-asm findings import' to see them here.")}
	}
	ranked := append([]service.FindingView{}, open...)
	sort.SliceStable(ranked, func(a, b int) bool { return ranked[a].Risk.Score > ranked[b].Risk.Score })
	limit := 5
	if len(ranked) < limit {
		limit = len(ranked)
	}
	var out []fyne.CanvasObject
	for _, f := range ranked[:limit] {
		text := fmt.Sprintf("%s  score %d  %s on %s\nWhy: %s", severityLabel(string(f.Severity)), f.Risk.Score, f.Title, f.AssetValue, strings.Join(f.Risk.Factors, "; "))
		out = append(out, wrapped(text))
	}
	return out
}

func staleAssets(snap snapshot) []fyne.CanvasObject {
	now := snap.Now
	lastSeen := map[string]time.Time{}
	for _, f := range snap.Findings {
		if f.LastSeen.After(lastSeen[f.AssetValue]) {
			lastSeen[f.AssetValue] = f.LastSeen
		}
	}
	var out []fyne.CanvasObject
	for _, asset := range snap.Assets {
		var reasons []string
		if !scopeAuthorizes(snap.Scopes, asset) {
			reasons = append(reasons, "outside every registered scope")
		}
		seen, ok := lastSeen[asset.Value]
		switch {
		case ok && now.Sub(seen) > staleAfterDays*24*time.Hour:
			reasons = append(reasons, fmt.Sprintf("no finding seen in %d days", staleAfterDays))
		case !ok && now.Sub(asset.CreatedAt) > staleAfterDays*24*time.Hour:
			reasons = append(reasons, "no finding recorded and added more than 30 days ago")
		}
		if len(reasons) > 0 {
			out = append(out, wrapped(fmt.Sprintf("%s: %s", asset.Value, strings.Join(reasons, "; "))))
		}
	}
	if len(out) == 0 {
		out = append(out, wrapped("None. Every asset is inside a scope and has a recent finding or was added recently."))
	}
	if len(out) > 10 {
		out = append(out[:10], wrapped(fmt.Sprintf("...and %d more on the Assets view.", len(out)-10)))
	}
	return out
}

func scopeAuthorizes(scopes []domain.Scope, asset domain.Asset) bool {
	for _, scope := range scopes {
		if scope.Authorizes(asset.Kind, asset.Value) {
			return true
		}
	}
	return false
}

func lastImport(snap snapshot) []fyne.CanvasObject {
	if len(snap.Batches) == 0 {
		return []fyne.CanvasObject{wrapped("No imports yet.")}
	}
	latest := snap.Batches[0]
	for _, b := range snap.Batches {
		if b.CreatedAt.After(latest.CreatedAt) {
			latest = b
		}
	}
	return []fyne.CanvasObject{wrapped(fmt.Sprintf("%s from %s (%s): %d accepted, %d created, %d updated.",
		stamp(latest.CreatedAt), latest.Source, latest.Format, latest.Accepted, latest.Created, latest.Updated))}
}

func (s *session) changesSummary(snap snapshot) []fyne.CanvasObject {
	if len(snap.Batches) == 0 {
		return []fyne.CanvasObject{wrapped("No imports yet.")}
	}
	latest := snap.Batches[0]
	for _, b := range snap.Batches {
		if b.CreatedAt.After(latest.CreatedAt) {
			latest = b
		}
	}
	var report service.ChangeReport
	err := s.withService(func(svc *service.Service, actor domain.Actor) error {
		var err error
		report, err = svc.Changes(s.ctx, actor, latest.Source)
		return err
	})
	if err != nil {
		return []fyne.CanvasObject{wrapped(err.Error())}
	}
	if report.Previous.ID == 0 {
		return []fyne.CanvasObject{wrapped(fmt.Sprintf("Only one import from %s so far, so there is nothing to compare.", latest.Source))}
	}
	return []fyne.CanvasObject{wrapped(fmt.Sprintf("Source %s compared with the previous import: %d new, %d missing, %d severity changed. Details are in the Changes view.",
		latest.Source, len(report.New), len(report.Missing), len(report.SeverityChanged)))}
}
