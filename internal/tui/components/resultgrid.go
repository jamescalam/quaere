package components

import (
	"fmt"
	"strings"

	"github.com/aurelio-labs/quaere/internal/db"
	"github.com/aurelio-labs/quaere/internal/tui/theme"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// maxColWidth caps how wide any single result column is allowed to grow.
const maxColWidth = 48

// ResultGrid displays the rows returned by the last query in a scrollable table,
// with a one-line summary (command tag, row count, elapsed time) above it.
type ResultGrid struct {
	base
	tbl     table.Model
	rs      *db.ResultSet
	info    string // summary line, or an error/help message
	isError bool
}

func NewResultGrid() *ResultGrid {
	t := table.New(table.WithFocused(false))
	t.SetStyles(gridStyles())
	return &ResultGrid{tbl: t, info: "Run a query (ctrl+r) or pick a table to see results."}
}

func gridStyles() table.Styles {
	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(theme.Border).
		BorderBottom(true).
		Bold(true).
		Foreground(theme.Accent)
	s.Selected = s.Selected.
		Foreground(lipgloss.Color("0")).
		Background(theme.Primary).
		Bold(false)
	return s
}

func (g *ResultGrid) Init() tea.Cmd { return nil }

// SetResult loads a successful result set into the grid.
func (g *ResultGrid) SetResult(rs *db.ResultSet) {
	g.rs = rs
	g.isError = false
	if len(rs.Columns) == 0 {
		// Non-SELECT statement: nothing to tabulate.
		g.tbl.SetColumns(nil)
		g.tbl.SetRows(nil)
		g.info = fmt.Sprintf("%s · %s", nonEmpty(rs.Command, "OK"), rs.Elapsed.Round(1e6))
		return
	}
	g.info = fmt.Sprintf("%s · %d row(s) · %s", nonEmpty(rs.Command, "OK"), len(rs.Rows), rs.Elapsed.Round(1e6))
	g.rebuildTable()
}

// SetError shows an error message in place of results.
func (g *ResultGrid) SetError(err error) {
	g.rs = nil
	g.isError = true
	g.info = err.Error()
	g.tbl.SetColumns(nil)
	g.tbl.SetRows(nil)
}

// SetInfo shows a neutral status message (e.g. "connecting…").
func (g *ResultGrid) SetInfo(msg string) {
	g.isError = false
	g.info = msg
}

func (g *ResultGrid) rebuildTable() {
	if g.rs == nil {
		return
	}
	widths := columnWidths(g.rs.Columns, g.rs.Rows)
	cols := make([]table.Column, len(g.rs.Columns))
	for i, name := range g.rs.Columns {
		cols[i] = table.Column{Title: name, Width: widths[i]}
	}
	rows := make([]table.Row, len(g.rs.Rows))
	for i, r := range g.rs.Rows {
		rows[i] = table.Row(sanitizeRow(r))
	}
	g.tbl.SetColumns(cols)
	g.tbl.SetRows(rows)
	g.tbl.GotoTop()
	g.resizeTable()
}

// columnWidths sizes each column to the widest of its header/values, capped.
func columnWidths(cols []string, rows [][]string) []int {
	w := make([]int, len(cols))
	for i, c := range cols {
		w[i] = lipgloss.Width(c)
	}
	for _, r := range rows {
		for i := range cols {
			if i < len(r) {
				if cw := lipgloss.Width(firstLine(r[i])); cw > w[i] {
					w[i] = cw
				}
			}
		}
	}
	for i := range w {
		if w[i] > maxColWidth {
			w[i] = maxColWidth
		}
		w[i]++ // a little breathing room
	}
	return w
}

// sanitizeRow collapses multi-line cell values to their first line so the grid
// stays one row per record.
func sanitizeRow(r []string) []string {
	out := make([]string, len(r))
	for i, v := range r {
		out[i] = firstLine(v)
	}
	return out
}

func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i != -1 {
		return s[:i] + "…"
	}
	return s
}

func (g *ResultGrid) Focus() {
	g.base.Focus()
	g.tbl.Focus()
}

func (g *ResultGrid) Blur() {
	g.base.Blur()
	g.tbl.Blur()
}

func (g *ResultGrid) SetSize(w, h int) {
	g.base.SetSize(w, h)
	g.resizeTable()
}

func (g *ResultGrid) resizeTable() {
	// Inner box height minus the info line; the table draws its own header row.
	bodyH := theme.InnerHeight(g.height) - 1
	if bodyH < 1 {
		bodyH = 1
	}
	g.tbl.SetHeight(bodyH)
	g.tbl.SetWidth(theme.InnerWidth(g.width))
}

func (g *ResultGrid) Update(msg tea.Msg) tea.Cmd {
	if !g.focused {
		return nil
	}
	var cmd tea.Cmd
	g.tbl, cmd = g.tbl.Update(msg)
	return cmd
}

func (g *ResultGrid) View() string {
	infoStyle := theme.MutedStyle
	if g.isError {
		infoStyle = lipgloss.NewStyle().Foreground(theme.Red)
	}
	info := infoStyle.Render(truncate(g.info, theme.InnerWidth(g.width)))

	var body string
	if g.rs != nil && len(g.rs.Columns) > 0 {
		body = lipgloss.JoinVertical(lipgloss.Left, info, g.tbl.View())
	} else {
		body = info
	}
	body = clipHeight(body, theme.InnerHeight(g.height))
	return theme.Box("Results", g.focused, g.width, g.height, body)
}

func nonEmpty(s, fallback string) string {
	if strings.TrimSpace(s) == "" {
		return fallback
	}
	return s
}
