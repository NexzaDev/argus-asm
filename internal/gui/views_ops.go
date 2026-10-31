//go:build !nogui

package gui

import (
	"fmt"
	"io"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/NexzaDev/argus-asm/internal/domain"
	"github.com/NexzaDev/argus-asm/internal/export"
	"github.com/NexzaDev/argus-asm/internal/service"
)

func buildNotifications(s *session) fyne.CanvasObject {
	filterOptions := []string{"Unread only", "All notifications"}
	filter := widget.NewSelect(filterOptions, nil)
	if s.notifyUnreadOnly {
		filter.SetSelected("Unread only")
	} else {
		filter.SetSelected("All notifications")
	}
	filter.OnChanged = func(choice string) {
		s.notifyUnreadOnly = choice == "Unread only"
		s.show(indexNotifications)
	}
	if s.notifySeverity == "" {
		s.notifySeverity = "All severities"
	}
	severityOptions := []string{"All severities", "Critical", "High", "Medium", "Low", "Info", "Not rated"}
	severities := widget.NewSelect(severityOptions, nil)
	severities.SetSelected(s.notifySeverity)
	severities.OnChanged = func(choice string) {
		s.notifySeverity = choice
		s.show(indexNotifications)
	}
	data := tableData{Columns: []string{"Time", "Severity", "Kind", "State", "Message"}, Widths: []float32{150, 120, 110, 90, 400}, HasSeverity: true, SeverityCol: 1, Empty: "No notifications. They appear after 'argus-asm notify refresh' finds ticket or certificate events."}
	var ids []int64
	for _, n := range s.snap.Notifications {
		if s.notifyUnreadOnly && n.AcknowledgedAt != nil {
			continue
		}
		sev, rated := domain.NotificationSeverity(n)
		if !matchesSeverityFilter(s.notifySeverity, sev, rated) {
			continue
		}
		state := "unread"
		if n.AcknowledgedAt != nil {
			state = "read"
		}
		severityValue := ""
		if rated {
			severityValue = string(sev)
		}
		data.Rows = append(data.Rows, []string{stamp(n.CreatedAt), severityValue, n.Kind, state, n.Message})
		data.Details = append(data.Details, fmt.Sprintf("Notification %d\nTime: %s\nKind: %s\nState: %s\nSeverity: %s\n\n%s", n.ID, stamp(n.CreatedAt), n.Kind, state, notificationSeverityText(sev, rated), n.Message))
		ids = append(ids, n.ID)
	}
	var table *tableView
	markRead := widget.NewButtonWithIcon("Mark as read", mustIcon("circle-check", theme.Color(theme.ColorNameForeground)), func() {
		if table == nil || table.selectedRow() < 0 || table.selectedRow() >= len(ids) {
			table.setSummary("Select a notification first.")
			return
		}
		id := ids[table.selectedRow()]
		err := s.withService(func(svc *service.Service, actor domain.Actor) error {
			return svc.Acknowledge(s.ctx, actor, id)
		})
		if err != nil {
			table.setSummary(err.Error())
			return
		}
		s.reload()
		s.show(indexNotifications)
	})
	table = newTableView(s, "Notifications", data, fitSelect(filter, filterOptions), fitSelect(severities, severityOptions), markRead)
	return table.object()
}

func buildReports(s *session) fyne.CanvasObject {
	status := wrapped("Exports are written only where you choose. They contain stored values, not credentials. Export cells that start with spreadsheet formula characters are neutralized.")
	status.Wrapping = fyne.TextWrapWord
	findings := widget.NewButtonWithIcon("Export all findings (CSV)", mustIcon("download", theme.Color(theme.ColorNameForeground)), func() {
		saveReport(s, status, func(w io.Writer) (int, error) {
			var views []service.FindingView
			err := s.withService(func(svc *service.Service, actor domain.Actor) error {
				var err error
				views, err = svc.ExportFindings(s.ctx, actor, true)
				return err
			})
			if err != nil {
				return 0, err
			}
			return len(views), export.WriteFindingsCSV(w, views)
		})
	})
	assets := widget.NewButtonWithIcon("Export assets (CSV)", mustIcon("download", theme.Color(theme.ColorNameForeground)), func() {
		saveReport(s, status, func(w io.Writer) (int, error) {
			var assets []domain.Asset
			err := s.withService(func(svc *service.Service, actor domain.Actor) error {
				var err error
				assets, err = svc.ExportAssets(s.ctx, actor)
				return err
			})
			if err != nil {
				return 0, err
			}
			return len(assets), export.WriteAssetsCSV(w, assets)
		})
	})
	return container.NewVScroll(container.NewVBox(widget.NewLabelWithStyle("Reports", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}), findings, assets, status))
}

func saveReport(s *session, status *flexLabel, write func(w io.Writer) (int, error)) {
	dialog.ShowFileSave(func(w fyne.URIWriteCloser, err error) {
		if err != nil {
			status.SetText(fmt.Sprintf("Export failed: %v", err))
			return
		}
		if w == nil {
			return
		}
		defer w.Close()
		count, err := write(w)
		if err != nil {
			status.SetText(fmt.Sprintf("Export failed: %v", err))
			return
		}
		status.SetText(fmt.Sprintf("Exported %d rows to %s.", count, w.URI().Name()))
	}, s.window)
}

func buildSettings(s *session) fyne.CanvasObject {
	status := widget.NewLabel("")
	status.Wrapping = fyne.TextWrapWord
	density := widget.NewRadioGroup([]string{"Comfortable", "Compact"}, nil)
	density.Horizontal = true
	if getDensity() == densityCompact {
		density.SetSelected("Compact")
	} else {
		density.SetSelected("Comfortable")
	}
	density.OnChanged = func(choice string) {
		if choice == "Compact" {
			setDensity(densityCompact)
		} else {
			setDensity(densityComfortable)
		}
		s.show(indexSettings)
	}
	forget := widget.NewButtonWithIcon("Forget remembered launcher choice", mustIcon("x", theme.Color(theme.ColorNameForeground)), func() {
		if s.req.Forget == nil {
			status.SetText("This build cannot change the launcher preference.")
			return
		}
		if err := s.req.Forget(); err != nil {
			status.SetText(fmt.Sprintf("Could not forget the choice: %v", err))
			return
		}
		status.SetText("Remembered launcher choice removed. The launcher asks again next time.")
	})
	reload := widget.NewButtonWithIcon("Reload stored data", mustIcon("refresh-cw", theme.Color(theme.ColorNameForeground)), func() {
		s.reload()
		s.show(indexSettings)
	})
	db := wrapped("Database file: " + displayName(s.req.DBPath))
	db.Wrapping = fyne.TextWrapWord
	appearance := widget.NewLabel("Colors follow the operating system light or dark setting. Severity uses an icon, the word, and a color.")
	appearance.Wrapping = fyne.TextWrapWord
	return container.NewVScroll(container.NewVBox(
		widget.NewLabelWithStyle("Table density", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		density,
		widget.NewLabelWithStyle("Appearance", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		appearance,
		widget.NewLabelWithStyle("Launcher", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		wrapped("Change the stored choice with 'argus-asm config set-ui' or remove it here."),
		forget,
		widget.NewLabelWithStyle("Data", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		db,
		reload,
		status,
	))
}

func displayName(path string) string {
	if path == "" {
		return "not configured"
	}
	parts := strings.FieldsFunc(path, func(r rune) bool { return r == '/' || r == '\\' })
	if len(parts) == 0 {
		return path
	}
	return parts[len(parts)-1]
}

func matchesSeverityFilter(choice string, sev domain.Severity, rated bool) bool {
	switch choice {
	case "", "All severities":
		return true
	case "Not rated":
		return !rated
	default:
		return rated && strings.EqualFold(choice, string(sev))
	}
}

func notificationSeverityText(sev domain.Severity, rated bool) string {
	if !rated {
		return "not rated"
	}
	return severityLabel(string(sev))
}
