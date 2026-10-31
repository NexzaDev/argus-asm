//go:build !nogui

package gui

import (
	"encoding/csv"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/NexzaDev/argus-asm/internal/domain"
	"github.com/NexzaDev/argus-asm/internal/export"
)

type tableData struct {
	Columns     []string
	Widths      []float32
	Rows        [][]string
	Keys        [][]string
	Details     []string
	HasSeverity bool
	SeverityCol int
	Empty       string
}

type tableView struct {
	s        *session
	name     string
	data     tableData
	order    []int
	selected int
	sortCol  int
	desc     bool
	query    string
	table    *widget.Table
	detail   *widget.Label
	summary  *widget.Label
	filter   *widget.Entry
	root     fyne.CanvasObject
}

func severityLabel(value string) string {
	if value == "" {
		return ""
	}
	return strings.ToUpper(value[:1]) + value[1:]
}

func fitSelect(sel *widget.Select, options []string) fyne.CanvasObject {
	return container.New(layout.NewGridWrapLayout(fyne.NewSize(selectWidthFor(options), sel.MinSize().Height)), sel)
}

func selectWidthFor(options []string) float32 {
	longest := float32(0)
	for _, option := range options {
		if w := fyne.MeasureText(option, theme.TextSize(), fyne.TextStyle{}).Width; w > longest {
			longest = w
		}
	}
	return longest + theme.Size(theme.SizeNameInnerPadding)*2 + theme.IconInlineSize() + theme.Size(theme.SizeNamePadding)*4
}

func newTableView(s *session, name string, data tableData, extra ...fyne.CanvasObject) *tableView {
	v := &tableView{s: s, name: name, data: data, selected: -1}
	if len(data.Widths) < len(data.Columns) {
		widths := make([]float32, len(data.Columns))
		for i := range widths {
			widths[i] = 170
		}
		v.data.Widths = widths
	}
	v.summary = widget.NewLabel("")
	v.detail = widget.NewLabel("Select a row to see its details here.")
	v.detail.Wrapping = fyne.TextWrapWord
	v.filter = widget.NewEntry()
	v.filter.SetPlaceHolder("Filter rows (Ctrl+F)")
	s.filter = v.filter
	sortColumns := append([]string{}, data.Columns...)
	sortSelect := widget.NewSelect(sortColumns, nil)
	sortSelect.SetSelectedIndex(0)
	sortSelect.OnChanged = func(name string) {
		for i, column := range v.data.Columns {
			if column == name {
				v.sortCol = i
			}
		}
		v.apply()
	}
	direction := widget.NewSelect([]string{"Ascending", "Descending"}, nil)
	direction.SetSelected("Ascending")
	direction.OnChanged = func(choice string) {
		v.desc = choice == "Descending"
		v.apply()
	}
	copyButton := widget.NewButtonWithIcon("Copy row", mustIcon("copy", theme.Color(theme.ColorNameForeground)), func() {
		if v.selected < 0 {
			v.summary.SetText("Select a row first.")
			return
		}
		fyne.CurrentApp().Clipboard().SetContent(v.rowText(v.selected))
		v.summary.SetText("Row copied to the clipboard.")
	})
	exportButton := widget.NewButtonWithIcon("Export CSV", mustIcon("download", theme.Color(theme.ColorNameForeground)), func() {
		v.export()
	})
	densityLabel := "Compact density"
	if getDensity() == densityCompact {
		densityLabel = "Comfortable density"
	}
	densityButton := widget.NewButton(densityLabel, func() {
		if getDensity() == densityCompact {
			setDensity(densityComfortable)
		} else {
			setDensity(densityCompact)
		}
		s.show(s.current)
	})
	actions := append([]fyne.CanvasObject{copyButton, exportButton, densityButton}, extra...)
	actions = append(actions, v.summary)
	toolbar := container.NewVBox(
		container.NewGridWithColumns(3, v.filter, sortSelect, direction),
		container.NewHBox(actions...),
	)

	v.table = widget.NewTable(
		func() (int, int) { return len(v.order) + 1, len(v.data.Columns) },
		v.newCell,
		v.updateCell,
	)
	for i, w := range v.data.Widths {
		v.table.SetColumnWidth(i, w)
	}
	v.filter.OnChanged = func(text string) {
		v.query = strings.ToLower(strings.TrimSpace(text))
		v.apply()
	}
	v.table.OnSelected = func(id widget.TableCellID) {
		if id.Row == 0 || id.Row-1 >= len(v.order) {
			return
		}
		v.selected = v.order[id.Row-1]
		v.detail.SetText(v.rowText(v.selected))
	}
	v.apply()
	detailPanel := container.NewVScroll(v.detail)
	split := container.NewHSplit(v.table, detailPanel)
	split.Offset = 0.66
	v.root = container.NewBorder(toolbar, nil, nil, nil, split)
	return v
}

func (v *tableView) object() fyne.CanvasObject {
	return v.root
}

func (v *tableView) selectedRow() int {
	return v.selected
}

func (v *tableView) setSummary(text string) {
	v.summary.SetText(text)
}

func (v *tableView) newCell() fyne.CanvasObject {
	return newTableCell()
}

func (v *tableView) updateCell(id widget.TableCellID, obj fyne.CanvasObject) {
	cell := obj.(*tableCell)
	cell.reset()
	if id.Row == 0 {
		cell.label.TextStyle = fyne.TextStyle{Bold: true}
		cell.label.SetText(v.data.Columns[id.Col])
		cell.label.Show()
		return
	}
	idx := v.order[id.Row-1]
	value := v.data.Rows[idx][id.Col]
	if v.data.HasSeverity && id.Col == v.data.SeverityCol && value == "" {
		cell.icon.Hide()
		cell.text.Hide()
		cell.label.SetText("not rated")
		cell.label.Show()
		return
	}
	if v.data.HasSeverity && id.Col == v.data.SeverityCol {
		severity := domain.Severity(value)
		if res, err := iconResource(severityIconName(severity), theme.Color(theme.ColorNameForeground)); err == nil {
			cell.icon.SetResource(res)
		}
		cell.icon.Show()
		cell.text.Text = severityLabel(value)
		cell.text.Color = severityTextColor(severity, currentVariant())
		cell.text.Show()
		return
	}
	cell.label.SetText(value)
	cell.label.Show()
}

type tableCell struct {
	widget.BaseWidget
	icon  *widget.Icon
	label *widget.Label
	text  *canvas.Text
	body  *fyne.Container
}

func newTableCell() *tableCell {
	c := &tableCell{icon: widget.NewIcon(nil), label: widget.NewLabel("Ag"), text: canvas.NewText("", theme.Color(theme.ColorNameForeground))}
	c.label.Truncation = fyne.TextTruncateEllipsis
	c.text.TextStyle = fyne.TextStyle{Bold: true}
	c.ExtendBaseWidget(c)
	c.body = container.NewBorder(nil, nil, c.icon, nil, container.NewStack(c.label, c.text))
	return c
}

func (c *tableCell) reset() {
	c.label.TextStyle = fyne.TextStyle{}
	c.label.SetText("")
	c.label.Show()
	c.icon.Hide()
	c.text.Hide()
}

func (c *tableCell) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(c.body)
}

func (v *tableView) rowText(idx int) string {
	if idx >= 0 && idx < len(v.data.Details) && v.data.Details[idx] != "" {
		return v.data.Details[idx]
	}
	var b strings.Builder
	for i, column := range v.data.Columns {
		fmt.Fprintf(&b, "%s: %s\n", column, v.data.Rows[idx][i])
	}
	return b.String()
}

func (v *tableView) apply() {
	order := make([]int, 0, len(v.data.Rows))
	for i, row := range v.data.Rows {
		if v.query != "" && !strings.Contains(strings.ToLower(strings.Join(row, " ")), v.query) {
			continue
		}
		order = append(order, i)
	}
	keyOf := func(idx int) string {
		if v.data.Keys != nil && v.sortCol < len(v.data.Keys[idx]) {
			return v.data.Keys[idx][v.sortCol]
		}
		return v.data.Rows[idx][v.sortCol]
	}
	sort.SliceStable(order, func(a, b int) bool {
		ka, kb := keyOf(order[a]), keyOf(order[b])
		fa, errA := strconv.ParseFloat(ka, 64)
		fb, errB := strconv.ParseFloat(kb, 64)
		less := strings.ToLower(ka) < strings.ToLower(kb)
		if errA == nil && errB == nil {
			less = fa < fb
		}
		if v.desc {
			return !less && ka != kb
		}
		return less
	})
	v.order = order
	v.summary.SetText(fmt.Sprintf("%d of %d rows", len(order), len(v.data.Rows)))
	if len(v.data.Rows) == 0 && v.data.Empty != "" {
		v.summary.SetText(v.data.Empty)
	}
	v.selected = -1
	v.detail.SetText("Select a row to see its details here.")
	v.table.UnselectAll()
	v.table.Refresh()
}

func (v *tableView) export() {
	rows := make([][]string, 0, len(v.order))
	for _, idx := range v.order {
		rows = append(rows, v.data.Rows[idx])
	}
	header := v.data.Columns
	dialog.ShowFileSave(func(w fyne.URIWriteCloser, err error) {
		if err != nil {
			v.summary.SetText(fmt.Sprintf("Export failed: %v", err))
			return
		}
		if w == nil {
			return
		}
		defer w.Close()
		if err := writeRowsCSV(w, header, rows); err != nil {
			v.summary.SetText(fmt.Sprintf("Export failed: %v", err))
			return
		}
		v.summary.SetText(fmt.Sprintf("Exported %d rows.", len(rows)))
	}, v.s.window)
}

func writeRowsCSV(w io.Writer, header []string, rows [][]string) error {
	writer := csv.NewWriter(w)
	safeHeader := make([]string, len(header))
	for i, h := range header {
		safeHeader[i] = export.Neutralize(h)
	}
	if err := writer.Write(safeHeader); err != nil {
		return err
	}
	for _, row := range rows {
		safe := make([]string, len(row))
		for i, cell := range row {
			safe[i] = export.Neutralize(cell)
		}
		if err := writer.Write(safe); err != nil {
			return err
		}
	}
	writer.Flush()
	return writer.Error()
}
