//go:build !nogui

package gui

import (
	"bytes"
	"context"
	"fmt"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/software"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/NexzaDev/argus-asm/internal/buildinfo"
	"github.com/NexzaDev/argus-asm/internal/config"
	"github.com/NexzaDev/argus-asm/internal/desktop"
	"github.com/NexzaDev/argus-asm/internal/domain"
	"github.com/NexzaDev/argus-asm/internal/repository/sqlite"
	"github.com/NexzaDev/argus-asm/internal/service"
)

var fixedNow = time.Date(2026, 10, 29, 9, 0, 0, 0, time.UTC)

func seedDatabase(t *testing.T) string {
	t.Helper()
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "argus.db")
	store, err := sqlite.Open(ctx, path, true)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	if err := store.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	svc := service.New(store, func() time.Time { return fixedNow })
	actor, err := svc.ResolveActor(ctx, "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.AddScope(ctx, actor, "example.test"); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.AddScope(ctx, actor, "192.0.2.0/24"); err != nil {
		t.Fatal(err)
	}
	for _, asset := range []string{"www.example.test", "api.example.test", "192.0.2.10"} {
		if _, err := svc.AddAsset(ctx, actor, asset); err != nil {
			t.Fatal(err)
		}
	}
	file, err := os.Open(filepath.Join("..", "..", "testdata", "demo", "findings.csv"))
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	if _, err := svc.ImportFindings(ctx, actor, "csv", "demo-scanner", file, false); err != nil {
		t.Fatal(err)
	}
	return path
}

func collectButtons(obj fyne.CanvasObject, into map[string]*widget.Button) {
	switch o := obj.(type) {
	case *widget.Button:
		into[o.Text] = o
	case *fyne.Container:
		for _, child := range o.Objects {
			collectButtons(child, into)
		}
	}
}

func findCheck(obj fyne.CanvasObject) *widget.Check {
	switch o := obj.(type) {
	case *widget.Check:
		return o
	case *fyne.Container:
		for _, child := range o.Objects {
			if found := findCheck(child); found != nil {
				return found
			}
		}
	}
	return nil
}

func TestUnavailableReportsMissingDisplay(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("display detection is only needed on Linux")
	}
	t.Setenv("DISPLAY", "")
	t.Setenv("WAYLAND_DISPLAY", "")
	if reason := (Runner{}).Unavailable(); !strings.Contains(reason, "DISPLAY") {
		t.Fatalf("reason = %q", reason)
	}
	t.Setenv("DISPLAY", ":0")
	if reason := (Runner{}).Unavailable(); reason != "" {
		t.Fatalf("display present but reason = %q", reason)
	}
}

func TestChooserTerminalRemembersConsole(t *testing.T) {
	test.NewApp()
	var remembered []string
	s := &session{
		app:    fyne.CurrentApp(),
		window: test.NewWindow(nil),
		req: desktop.Request{ShowChooser: true, Remember: func(mode string) error {
			remembered = append(remembered, mode)
			return nil
		}},
	}
	content := s.chooser()
	buttons := map[string]*widget.Button{}
	collectButtons(content, buttons)
	check := findCheck(content)
	if check == nil {
		t.Fatal("remember checkbox missing")
	}
	check.SetChecked(true)
	if buttons["Terminal"] == nil || buttons["Dashboard (graphical window)"] == nil {
		t.Fatalf("chooser buttons missing: %v", buttons)
	}
	test.Tap(buttons["Terminal"])
	if s.result.Choice != config.UIConsole {
		t.Fatalf("choice = %q", s.result.Choice)
	}
	if len(remembered) != 1 || remembered[0] != config.UIConsole {
		t.Fatalf("remembered = %v", remembered)
	}
}

func TestChooserDashboardRemembersGUI(t *testing.T) {
	test.NewApp()
	var remembered []string
	window := test.NewWindow(nil)
	s := &session{
		ctx:    context.Background(),
		app:    fyne.CurrentApp(),
		window: window,
		req: desktop.Request{DBPath: seedDatabase(t), Now: func() time.Time { return fixedNow }, ShowChooser: true, Remember: func(mode string) error {
			remembered = append(remembered, mode)
			return nil
		}},
	}
	content := s.chooser()
	buttons := map[string]*widget.Button{}
	collectButtons(content, buttons)
	check := findCheck(content)
	if check == nil {
		t.Fatal("remember checkbox missing")
	}
	check.SetChecked(true)
	test.Tap(buttons["Dashboard (graphical window)"])
	if s.result.Choice != config.UIGUI {
		t.Fatalf("choice = %q", s.result.Choice)
	}
	if len(remembered) != 1 || remembered[0] != config.UIGUI {
		t.Fatalf("remembered = %v", remembered)
	}
}

func TestOverviewCountsComeFromStoredData(t *testing.T) {
	path := seedDatabase(t)
	snap, err := loadSnapshot(context.Background(), desktop.Request{DBPath: path, Now: func() time.Time { return fixedNow }})
	if err != nil {
		t.Fatal(err)
	}
	if len(snap.Assets) != 3 || len(snap.Scopes) != 2 {
		t.Fatalf("assets=%d scopes=%d", len(snap.Assets), len(snap.Scopes))
	}
	if len(openFindings(snap)) == 0 {
		t.Fatal("expected open findings from the demo CSV")
	}
	if !snap.Demo {
		t.Fatal("documentation names should be labeled as demo data")
	}
}

func TestDashboardReportsMissingDatabase(t *testing.T) {
	_, err := loadSnapshot(context.Background(), desktop.Request{DBPath: filepath.Join(t.TempDir(), "missing.db")})
	if err == nil || !strings.Contains(err.Error(), "argus-asm init") {
		t.Fatalf("err = %v", err)
	}
}

func TestTierBoundaries(t *testing.T) {
	cases := map[int]string{100: "Critical", 90: "Critical", 89: "High", 70: "High", 69: "Medium", 40: "Medium", 39: "Low", 0: "Low"}
	for score, want := range cases {
		if got := tierOf(score); got != want {
			t.Errorf("tierOf(%d) = %s, want %s", score, got, want)
		}
	}
}

func relativeLuminance(c interface {
	RGBA() (uint32, uint32, uint32, uint32)
}) float64 {
	r, g, b, _ := c.RGBA()
	channel := func(v uint32) float64 {
		x := float64(v>>8) / 255
		if x <= 0.03928 {
			return x / 12.92
		}
		return math.Pow((x+0.055)/1.055, 2.4)
	}
	return 0.2126*channel(r) + 0.7152*channel(g) + 0.0722*channel(b)
}

func contrastRatio(a, b interface {
	RGBA() (uint32, uint32, uint32, uint32)
}) float64 {
	la, lb := relativeLuminance(a), relativeLuminance(b)
	if la < lb {
		la, lb = lb, la
	}
	return (la + 0.05) / (lb + 0.05)
}

func TestSeverityColorsMeetWCAGAA(t *testing.T) {
	light := theme.DefaultTheme()
	dark := theme.DefaultTheme()
	lightBG := light.Color(theme.ColorNameBackground, theme.VariantLight)
	darkBG := dark.Color(theme.ColorNameBackground, theme.VariantDark)
	for _, severity := range []domain.Severity{domain.SeverityCritical, domain.SeverityHigh, domain.SeverityMedium, domain.SeverityLow, domain.SeverityInfo} {
		if r := contrastRatio(severityTextColor(severity, theme.VariantLight), lightBG); r < 4.5 {
			t.Errorf("%s on light background: %.2f:1", severity, r)
		}
		if r := contrastRatio(severityTextColor(severity, theme.VariantDark), darkBG); r < 4.5 {
			t.Errorf("%s on dark background: %.2f:1", severity, r)
		}
	}
}

func TestEveryIconResolvesWithColor(t *testing.T) {
	names := map[string]bool{}
	for _, cat := range categories {
		names[cat.icon] = true
	}
	for _, name := range []string{"bug", "circle-check", "code", "copy", "download", "panel-left-close", "refresh-cw", "x", "octagon-alert", "triangle-alert", "circle-alert", "circle-dot", "info"} {
		names[name] = true
	}
	for name := range names {
		res, err := iconResource(name, theme.DefaultTheme().Color(theme.ColorNameForeground, theme.VariantDark))
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if strings.Contains(string(res.Content()), "currentColor") {
			t.Fatalf("%s still has currentColor", name)
		}
	}
}

func TestExternalLinksAllowOnlyTheProject(t *testing.T) {
	allowed := []string{
		"https://github.com/NexzaDev/argus-asm",
		"https://github.com/NexzaDev/argus-asm/issues/new?title=Bug",
	}
	for _, raw := range allowed {
		if _, err := projectLink(raw); err != nil {
			t.Errorf("%s rejected: %v", raw, err)
		}
	}
	denied := []string{
		"http://github.com/NexzaDev/argus-asm",
		"https://github.com/NexzaDev/argus-asm-evil",
		"https://github.com/someone/else",
		"https://example.com/NexzaDev/argus-asm",
		"https://user@github.com/NexzaDev/argus-asm",
		"file:///etc/passwd",
	}
	for _, raw := range denied {
		if _, err := projectLink(raw); err == nil {
			t.Errorf("%s accepted", raw)
		}
	}
}

func TestBugReportLinkIsPrefilledWithoutAttachments(t *testing.T) {
	body := bugReportBody()
	if !strings.Contains(body, "No logs, evidence, or credentials are attached") {
		t.Fatalf("body = %q", body)
	}
	link := newIssueURL(body)
	if !strings.HasPrefix(link, buildinfo.ProjectURL+"/issues/new?") {
		t.Fatalf("link = %s", link)
	}
	if _, err := projectLink(link); err != nil {
		t.Fatalf("issue link rejected: %v", err)
	}
	if strings.Contains(link, "\n") {
		t.Fatal("raw newline in link")
	}
}

func TestDocsCopiesMatchRepository(t *testing.T) {
	pairs := map[string]string{
		"docs/USAGE.md":               filepath.Join("..", "..", "docs", "USAGE.md"),
		"docs/THIRD_PARTY_NOTICES.md": filepath.Join("..", "..", "THIRD_PARTY_NOTICES.md"),
	}
	embedded := map[string]string{"docs/USAGE.md": usageGuide, "docs/THIRD_PARTY_NOTICES.md": thirdPartyNotices}
	for local, repo := range pairs {
		want, err := os.ReadFile(repo)
		if err != nil {
			t.Fatal(err)
		}
		got, err := os.ReadFile(local)
		if err != nil {
			t.Fatal(err)
		}
		if string(want) != string(got) || string(want) != embedded[local] {
			t.Fatalf("%s is out of date; copy %s into internal/gui/%s", local, repo, local)
		}
	}
}

func TestUsageSearchFiltersSections(t *testing.T) {
	blocks := usageBlocks()
	if len(blocks) < 5 {
		t.Fatalf("only %d usage sections", len(blocks))
	}
	matched := 0
	for _, block := range blocks {
		if strings.Contains(strings.ToLower(block), "scope add") {
			matched++
		}
	}
	if matched == 0 {
		t.Fatal("expected at least one usage section to mention scope add")
	}
}

func TestDiagnosticsOmitHomeDirectory(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		t.Skip("no home directory")
	}
	s := &session{req: desktop.Request{DBPath: filepath.Join(home, "data", "argus.db")}}
	text := diagnosticsText(s)
	if strings.Contains(text, home) {
		t.Fatalf("diagnostics leak the home directory: %s", text)
	}
	if !strings.Contains(text, "Database file: argus.db") {
		t.Fatalf("diagnostics = %s", text)
	}
}

func TestDemoLabelOnlyForDocumentationNames(t *testing.T) {
	for value, want := range map[string]bool{
		"www.example.test": true, "api.example.com": true, "192.0.2.10": true, "203.0.113.4": true,
		"intranet.corp.local": false, "8.8.8.8": false,
	} {
		if got := isDemoValue(value); got != want {
			t.Errorf("isDemoValue(%s) = %v, want %v", value, got, want)
		}
	}
}

func TestTableFiltersAndSorts(t *testing.T) {
	test.NewApp()
	s := &session{app: fyne.CurrentApp(), window: test.NewWindow(nil)}
	data := tableData{
		Columns:     []string{"Name", "Score"},
		Rows:        [][]string{{"beta", "10"}, {"alpha", "2"}, {"gamma", "30"}},
		Keys:        [][]string{{"b", "010"}, {"a", "002"}, {"g", "030"}},
		Details:     []string{"", "", ""},
		HasSeverity: false,
	}
	v := newTableView(s, "Test", data)
	if got := v.order; len(got) != 3 || v.data.Rows[got[0]][0] != "alpha" {
		t.Fatalf("default order = %v", got)
	}
	v.sortCol = 1
	v.desc = true
	v.apply()
	if v.data.Rows[v.order[0]][0] != "gamma" {
		t.Fatalf("descending score order starts with %s", v.data.Rows[v.order[0]][0])
	}
	v.query = "alp"
	v.apply()
	if len(v.order) != 1 || v.data.Rows[v.order[0]][0] != "alpha" {
		t.Fatalf("filtered order = %v", v.order)
	}
}

func TestShellBuildsEveryCategory(t *testing.T) {
	test.NewApp()
	fyne.CurrentApp().Settings().SetTheme(newArgusTheme())
	path := seedDatabase(t)
	s := &session{ctx: context.Background(), app: fyne.CurrentApp(), window: test.NewWindow(nil), req: desktop.Request{DBPath: path, Now: func() time.Time { return fixedNow }, Actor: ""}}
	s.shell()
	for i := range categories {
		s.show(i)
		if s.current != i {
			t.Fatalf("current = %d, want %d", s.current, i)
		}
	}
	s.setCollapsed(true)
	s.setCollapsed(false)
}

func TestWriteDesignScreenshots(t *testing.T) {
	dir := os.Getenv("ARGUS_SCREENSHOT_DIR")
	if dir == "" {
		t.Skip("set ARGUS_SCREENSHOT_DIR to write design screenshots")
	}
	test.NewApp()
	fyne.CurrentApp().Settings().SetTheme(newArgusTheme())
	path := seedDatabase(t)
	s := &session{ctx: context.Background(), app: fyne.CurrentApp(), window: test.NewWindow(nil), req: desktop.Request{DBPath: path, Now: func() time.Time { return fixedNow }}}
	content := s.shell()
	size := fyne.NewSize(1080, 700)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	for i, cat := range categories {
		s.show(i)
		canvas := software.NewCanvas()
		canvas.SetContent(content)
		canvas.Resize(size)
		name := fmt.Sprintf("%02d-%s.png", i+1, strings.ToLower(strings.ReplaceAll(cat.title, " ", "-")))
		var buf bytes.Buffer
		if err := png.Encode(&buf, canvas.Capture()); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, name), buf.Bytes(), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func TestShortcutTableIsConsistent(t *testing.T) {
	test.NewApp()
	table := shortcutTable()
	if len(table) != 15 {
		t.Fatalf("shortcut table has %d entries, want 15", len(table))
	}
	seen := map[string]bool{}
	labels := map[string]bool{}
	for _, sc := range table {
		if sc.label == "" || sc.desc == "" || sc.key == "" || sc.run == nil {
			t.Fatalf("incomplete shortcut %+v", sc)
		}
		id := fmt.Sprintf("%d/%s", sc.mod, sc.key)
		if seen[id] {
			t.Fatalf("duplicate binding %s", id)
		}
		if labels[sc.label] {
			t.Fatalf("duplicate label %s", sc.label)
		}
		seen[id] = true
		labels[sc.label] = true
	}
	tab, ok := shortcutsTab().(*fyne.Container)
	if !ok {
		t.Fatal("shortcuts tab is not a container")
	}
	if len(tab.Objects) != len(table)+1 {
		t.Fatalf("shortcuts tab has %d rows, want %d bindings plus the platform note", len(tab.Objects), len(table))
	}
	s := &session{window: test.NewWindow(nil)}
	s.registerShortcuts()
}

func TestAppIconIsEmbeddedPNG(t *testing.T) {
	img, err := png.Decode(bytes.NewReader(appIcon.Content()))
	if err != nil {
		t.Fatal(err)
	}
	if b := img.Bounds(); b.Dx() != 512 || b.Dy() != 512 {
		t.Fatalf("app icon is %dx%d, want 512x512", b.Dx(), b.Dy())
	}
}

func TestSeverityFilterMatchesRatedAndUnrated(t *testing.T) {
	cases := []struct {
		choice string
		sev    domain.Severity
		rated  bool
		want   bool
	}{
		{"All severities", domain.SeverityLow, false, true},
		{"Critical", domain.SeverityCritical, true, true},
		{"Critical", domain.SeverityHigh, true, false},
		{"High", domain.SeverityCritical, false, false},
		{"Not rated", "", false, true},
		{"Not rated", domain.SeverityInfo, true, false},
	}
	for _, tc := range cases {
		if got := matchesSeverityFilter(tc.choice, tc.sev, tc.rated); got != tc.want {
			t.Fatalf("matchesSeverityFilter(%q,%q,%v) = %v", tc.choice, tc.sev, tc.rated, got)
		}
	}
}

func TestNotificationFilterLabelsFitTheirSelects(t *testing.T) {
	test.NewApp()
	for _, options := range [][]string{
		{"Unread only", "All notifications"},
		{"All severities", "Critical", "High", "Medium", "Low", "Info", "Not rated"},
	} {
		sel := widget.NewSelect(options, nil)
		fitted := fitSelect(sel, options)
		for _, option := range options {
			need := fyne.MeasureText(option, theme.TextSize(), fyne.TextStyle{}).Width + 2*theme.Size(theme.SizeNamePadding) + theme.IconInlineSize()
			if got := fitted.MinSize().Width; got < need {
				t.Fatalf("select for %q is %.0f wide, needs at least %.0f for the label and arrow", option, got, need)
			}
		}
	}
}
