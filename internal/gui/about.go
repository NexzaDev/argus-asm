//go:build !nogui

package gui

import (
	"bufio"
	_ "embed"
	"fmt"
	"os"
	"runtime"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/NexzaDev/argus-asm/internal/buildinfo"
)

//go:embed docs/USAGE.md
var usageGuide string

//go:embed docs/THIRD_PARTY_NOTICES.md
var thirdPartyNotices string

//go:embed icons/LICENSE-lucide.txt
var lucideLicense string

func buildAbout(s *session) fyne.CanvasObject {
	tabs := container.NewAppTabs(
		container.NewTabItem("Overview", container.NewScroll(aboutOverview(s))),
		container.NewTabItem("Usage", usageTab()),
		container.NewTabItem("Shortcuts", container.NewScroll(shortcutsTab())),
		container.NewTabItem("Diagnostics", diagnosticsTab(s)),
		container.NewTabItem("Licenses", container.NewScroll(licensesTab())),
	)
	tabs.SetTabLocation(container.TabLocationTop)
	return tabs
}

func aboutOverview(s *session) fyne.CanvasObject {
	logo := canvas.NewImageFromResource(appIcon)
	logo.FillMode = canvas.ImageFillContain
	logo.SetMinSize(fyne.NewSize(96, 96))
	name := widget.NewLabelWithStyle(buildinfo.Name, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	lines := wrapped(fmt.Sprintf("%s\nVersion %s, license %s.\nSource: %s", buildinfo.Description, buildinfo.Version, buildinfo.License, buildinfo.ProjectURL))
	source := widget.NewButtonWithIcon("Source Code", mustIcon("code", theme.Color(theme.ColorNameForeground)), func() {
		openProject(s, buildinfo.ProjectURL)
	})
	bug := widget.NewButtonWithIcon("Found a bug? Please report it on GitHub.", mustIcon("bug", theme.Color(theme.ColorNameForeground)), func() {
		previewBugReport(s)
	})
	note := container.NewVBox(
		widget.NewLabel("Links open only when you click them."),
		widget.NewLabel("Nothing is sent automatically."),
		widget.NewLabel("The bug report preview lists everything that would be opened."),
	)
	return container.NewVBox(container.NewBorder(nil, nil, logo, nil, container.NewVBox(name, lines)), source, bug, note)
}

func openProject(s *session, raw string) {
	u, err := projectLink(raw)
	if err != nil {
		dialog.ShowError(err, s.window)
		return
	}
	if err := fyne.CurrentApp().OpenURL(u); err != nil {
		dialog.ShowError(err, s.window)
	}
}

func previewBugReport(s *session) {
	body := bugReportBody()
	preview := widget.NewLabel(body)
	preview.Wrapping = fyne.TextWrapWord
	content := container.NewVScroll(preview)
	content.SetMinSize(fyne.NewSize(480, 260))
	d := dialog.NewCustomConfirm("Bug report preview", "Open GitHub issue form", "Cancel", content, func(ok bool) {
		if !ok {
			return
		}
		openProject(s, newIssueURL(body))
	}, s.window)
	d.Show()
}

func usageTab() fyne.CanvasObject {
	blocks := usageBlocks()
	results := container.NewVBox()
	search := widget.NewEntry()
	search.SetPlaceHolder("Search the usage guide")
	render := func(query string) {
		results.Objects = nil
		q := strings.ToLower(strings.TrimSpace(query))
		count := 0
		for _, block := range blocks {
			if q != "" && !strings.Contains(strings.ToLower(block), q) {
				continue
			}
			results.Add(widget.NewRichTextFromMarkdown(block))
			count++
		}
		if count == 0 {
			results.Add(wrapped("No matching sections."))
		}
		results.Refresh()
	}
	search.OnChanged = render
	render("")
	return container.NewBorder(search, nil, nil, nil, container.NewScroll(results))
}

func usageBlocks() []string {
	var blocks []string
	var current strings.Builder
	scanner := bufio.NewScanner(strings.NewReader(usageGuide))
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "" {
			if current.Len() > 0 {
				blocks = append(blocks, current.String())
				current.Reset()
			}
			continue
		}
		current.WriteString(line)
		current.WriteString("\n")
	}
	if current.Len() > 0 {
		blocks = append(blocks, current.String())
	}
	return blocks
}

func shortcutsTab() fyne.CanvasObject {
	rows := container.NewVBox()
	for _, sc := range shortcutTable() {
		keys := widget.NewLabelWithStyle(sc.label, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
		rows.Add(container.NewBorder(nil, nil, keys, nil, wrapped(sc.desc)))
	}
	rows.Add(wrapped("On macOS the Ctrl modifier is Cmd."))
	return rows
}

func diagnosticsTab(s *session) fyne.CanvasObject {
	preview := wrapped(diagnosticsText(s))
	copyButton := widget.NewButtonWithIcon("Copy diagnostics", mustIcon("copy", theme.Color(theme.ColorNameForeground)), func() {
		fyne.CurrentApp().Clipboard().SetContent(preview.Text)
	})
	intro := container.NewVBox(
		widget.NewLabel("This is the complete diagnostics text."),
		widget.NewLabel("It contains no paths beyond the database file name, no evidence, and no credentials."),
		widget.NewLabel("It is sent nowhere unless you paste it yourself."),
	)
	return container.NewBorder(container.NewVBox(intro, copyButton), nil, nil, nil, container.NewScroll(preview))
}

func diagnosticsText(s *session) string {
	var b strings.Builder
	fmt.Fprintf(&b, "argus-asm diagnostics\n")
	fmt.Fprintf(&b, "Version: %s\n", buildinfo.Version)
	fmt.Fprintf(&b, "Commit: %s\n", buildinfo.Commit)
	fmt.Fprintf(&b, "Build date: %s\n", buildinfo.BuildDate)
	fmt.Fprintf(&b, "Platform: %s/%s\n", runtime.GOOS, runtime.GOARCH)
	fmt.Fprintf(&b, "Go runtime: %s\n", runtime.Version())
	fmt.Fprintf(&b, "Database file: %s\n", sanitizePath(displayName(s.req.DBPath)))
	if s.loadErr != nil {
		fmt.Fprintf(&b, "Database status: %s\n", sanitizePath(s.loadErr.Error()))
	} else {
		fmt.Fprintf(&b, "Database status: readable\n")
		fmt.Fprintf(&b, "Scopes: %d\nAssets: %d\nFindings: %d\n", len(s.snap.Scopes), len(s.snap.Assets), len(s.snap.Findings))
	}
	return b.String()
}

func sanitizePath(text string) string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return text
	}
	return strings.ReplaceAll(text, home, "~")
}

func licensesTab() fyne.CanvasObject {
	text := wrapped(thirdPartyNotices + "\n\nLucide icons (ISC and MIT, pinned lucide-static 1.64.0):\n\n" + lucideLicense)
	return text
}
