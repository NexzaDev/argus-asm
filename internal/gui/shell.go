//go:build !nogui

package gui

import (
	"context"
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	fyneDesktop "fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/NexzaDev/argus-asm/internal/desktop"
	"github.com/NexzaDev/argus-asm/internal/domain"
	"github.com/NexzaDev/argus-asm/internal/service"
)

type category struct {
	title string
	icon  string
	build func(s *session) fyne.CanvasObject
}

var categories []category

func init() {
	categories = []category{
		{"Overview", "layout-dashboard", buildOverview},
		{"Scope", "target", buildScope},
		{"Assets", "server", buildAssets},
		{"Findings", "bug", buildFindings},
		{"Imports", "upload", buildImports},
		{"Changes", "git-compare", buildChanges},
		{"Notifications", "bell", buildNotifications},
		{"Access", "users", buildAccess},
		{"Audit", "scroll-text", buildAudit},
		{"Reports", "file-output", buildReports},
		{"Settings", "settings", buildSettings},
		{"About", "info", buildAbout},
	}
}

const (
	indexNotifications = 6
	indexSettings      = 10
	indexAbout         = 11
)

type session struct {
	ctx              context.Context
	app              fyne.App
	window           fyne.Window
	req              desktop.Request
	result           desktop.Result
	snap             snapshot
	loadErr          error
	current          int
	body             *fyne.Container
	header           *widget.Label
	status           *widget.Label
	filter           *widget.Entry
	split            *container.Split
	buttons          []*navButton
	collapsed        bool
	changesBody      *fyne.Container
	notifyUnreadOnly bool
	notifySeverity   string
	bell             *widget.Button
	shellReady       bool
}

func (s *session) reload() {
	s.snap, s.loadErr = loadSnapshot(s.ctx, s.req)
}

func (s *session) withService(fn func(svc *service.Service, actor domain.Actor) error) error {
	return withService(s.ctx, s.req, fn)
}

func (s *session) shell() fyne.CanvasObject {
	s.reload()
	s.header = widget.NewLabelWithStyle("", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	s.status = widget.NewLabel("")
	s.body = container.NewStack()
	s.buttons = nil
	nav := make([]fyne.CanvasObject, 0, len(categories))
	for i, cat := range categories {
		index := i
		res := mustIcon(cat.icon, theme.Color(theme.ColorNameForeground))
		b := newNavButton(cat.title, res, func() { s.show(index) }, func(in bool) {
			if in {
				s.status.SetText(cat.title)
			} else {
				s.status.SetText(s.currentTitle())
			}
		})
		s.buttons = append(s.buttons, b)
		nav = append(nav, b)
	}
	collapse := widget.NewButtonWithIcon("", mustIcon("panel-left-close", theme.Color(theme.ColorNameForeground)), func() {
		s.setCollapsed(!s.collapsed)
	})
	sidebar := container.NewBorder(nil, collapse, nil, nil, container.NewVBox(nav...))
	s.split = container.NewHSplit(sidebar, container.NewBorder(s.header, nil, nil, nil, s.body))
	s.split.Offset = 0.2
	s.collapsed = false
	s.registerShortcuts()
	s.show(s.current)
	s.shellReady = true
	s.bell = widget.NewButtonWithIcon("0 unread", mustIcon("bell", theme.Color(theme.ColorNameForeground)), func() { s.show(indexNotifications) })
	s.bell.Importance = widget.LowImportance
	s.updateBell()
	return container.NewBorder(nil, container.NewBorder(nil, nil, nil, s.bell, s.status), nil, nil, s.split)
}

func (s *session) currentTitle() string {
	if s.current < 0 || s.current >= len(categories) {
		return ""
	}
	return categories[s.current].title
}

func (s *session) setCollapsed(collapsed bool) {
	s.collapsed = collapsed
	for i, b := range s.buttons {
		if collapsed {
			b.SetText("")
		} else {
			b.SetText(categories[i].title)
		}
		b.Refresh()
	}
	if collapsed {
		s.split.Offset = 0.06
	} else {
		s.split.Offset = 0.2
	}
	s.split.Refresh()
}

func (s *session) show(index int) {
	if index < 0 || index >= len(categories) {
		return
	}
	s.current = index
	s.filter = nil
	if s.header == nil || s.body == nil {
		return
	}
	s.header.SetText(categories[index].title + "    " + s.dataLabel())
	s.body.Objects = []fyne.CanvasObject{categories[index].build(s)}
	s.body.Refresh()
	for i, b := range s.buttons {
		if i == index {
			b.Importance = widget.HighImportance
		} else {
			b.Importance = widget.LowImportance
		}
		b.Refresh()
	}
	s.status.SetText(categories[index].title)
	s.updateBell()
}

func (s *session) updateBell() {
	if s.bell == nil {
		return
	}
	s.bell.SetText(fmt.Sprintf("%d unread", unreadCount(s.snap.Notifications)))
	s.bell.Refresh()
}

func (s *session) dataLabel() string {
	if s.loadErr != nil {
		return ""
	}
	if s.snap.Demo {
		return "[Demo data: synthetic names and documentation-only addresses]"
	}
	return "[Stored data]"
}

type shortcut struct {
	label string
	desc  string
	key   fyne.KeyName
	mod   fyne.KeyModifier
	run   func(*session)
}

func shortcutTable() []shortcut {
	mod := fyne.KeyModifierShortcutDefault
	digits := []fyne.KeyName{fyne.Key1, fyne.Key2, fyne.Key3, fyne.Key4, fyne.Key5, fyne.Key6, fyne.Key7, fyne.Key8, fyne.Key9, fyne.Key0}
	labels := []string{"1", "2", "3", "4", "5", "6", "7", "8", "9", "0"}
	var list []shortcut
	for i := range digits {
		index := i
		list = append(list, shortcut{
			label: "Ctrl+" + labels[i],
			desc:  categories[i].title,
			key:   digits[i],
			mod:   mod,
			run:   func(s *session) { s.show(index) },
		})
	}
	return append(list,
		shortcut{label: "Ctrl+,", desc: "Settings", key: fyne.KeyComma, mod: mod, run: func(s *session) { s.show(indexSettings) }},
		shortcut{label: "F1", desc: "About", key: fyne.KeyF1, mod: 0, run: func(s *session) { s.show(indexAbout) }},
		shortcut{label: "Ctrl+B", desc: "Collapse or expand the sidebar", key: fyne.KeyB, mod: mod, run: func(s *session) { s.setCollapsed(!s.collapsed) }},
		shortcut{label: "Ctrl+F", desc: "Focus the filter of the current table", key: fyne.KeyF, mod: mod, run: func(s *session) {
			if s.filter != nil {
				s.window.Canvas().Focus(s.filter)
			}
		}},
		shortcut{label: "Ctrl+R", desc: "Reload stored data", key: fyne.KeyR, mod: mod, run: func(s *session) {
			s.reload()
			s.show(s.current)
		}},
	)
}

func (s *session) registerShortcuts() {
	canvas := s.window.Canvas()
	for _, sc := range shortcutTable() {
		sc := sc
		canvas.AddShortcut(&fyneDesktop.CustomShortcut{KeyName: sc.key, Modifier: sc.mod}, func(fyne.Shortcut) { sc.run(s) })
	}
}

type navButton struct {
	widget.Button
	onHover func(bool)
}

func newNavButton(title string, res fyne.Resource, tapped func(), hover func(bool)) *navButton {
	b := &navButton{onHover: hover}
	b.ExtendBaseWidget(b)
	b.SetText(title)
	b.SetIcon(res)
	b.Alignment = widget.ButtonAlignLeading
	b.Importance = widget.LowImportance
	b.OnTapped = tapped
	return b
}

func (b *navButton) MouseIn(e *fyneDesktop.MouseEvent) {
	b.Button.MouseIn(e)
	if b.onHover != nil {
		b.onHover(true)
	}
}

func (b *navButton) MouseMoved(e *fyneDesktop.MouseEvent) {
	b.Button.MouseMoved(e)
}

func (b *navButton) MouseOut() {
	b.Button.MouseOut()
	if b.onHover != nil {
		b.onHover(false)
	}
}

func currentVariant() fyne.ThemeVariant {
	return fyne.CurrentApp().Settings().ThemeVariant()
}

func severityIconName(severity domain.Severity) string {
	switch severity {
	case domain.SeverityCritical:
		return "octagon-alert"
	case domain.SeverityHigh:
		return "triangle-alert"
	case domain.SeverityMedium:
		return "circle-alert"
	case domain.SeverityLow:
		return "circle-dot"
	}
	return "info"
}

func (s *session) errorView(err error) fyne.CanvasObject {
	message := widget.NewLabel(err.Error())
	message.Wrapping = fyne.TextWrapWord
	return container.NewPadded(message)
}

func newSession(ctx context.Context, a fyne.App, w fyne.Window, req desktop.Request) *session {
	return &session{ctx: ctx, app: a, window: w, req: req}
}
