package components

import (
	"strings"

	"github.com/aurelio-labs/quaere/internal/config"
	"github.com/aurelio-labs/quaere/internal/db"
	"github.com/aurelio-labs/quaere/internal/tui/messages"
	"github.com/aurelio-labs/quaere/internal/tui/theme"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type itemKind int

const (
	kindHeader itemKind = iota
	kindConn
	kindTable
)

type sidebarItem struct {
	kind  itemKind
	label string
	conn  config.Connection // when kindConn
	table db.Table          // when kindTable
}

func (i sidebarItem) selectable() bool { return i.kind != kindHeader }

// Sidebar lists saved connections and, once connected, the tables of the active
// database. Enter opens a connection or runs a table; n/e/d manage connections.
type Sidebar struct {
	base
	items      []sidebarItem
	cursor     int // index into items; kept on a selectable row
	conns      []config.Connection
	tables     []db.Table
	activeConn string // name of the currently connected connection (for marker)
}

func NewSidebar(conns []config.Connection) *Sidebar {
	s := &Sidebar{conns: conns}
	s.rebuild()
	return s
}

func (s *Sidebar) Init() tea.Cmd { return nil }

// SetConnections replaces the saved-connection list and rebuilds the tree.
func (s *Sidebar) SetConnections(conns []config.Connection) {
	s.conns = conns
	s.rebuild()
}

// SetTables replaces the table list (call with nil to clear on disconnect).
func (s *Sidebar) SetTables(tables []db.Table) {
	s.tables = tables
	s.rebuild()
}

// SetActive marks which connection is currently open.
func (s *Sidebar) SetActive(name string) { s.activeConn = name }

// rebuild recomputes the flat item list from connections + tables, keeping the
// cursor on a selectable row.
func (s *Sidebar) rebuild() {
	var items []sidebarItem
	items = append(items, sidebarItem{kind: kindHeader, label: "CONNECTIONS"})
	for _, c := range s.conns {
		items = append(items, sidebarItem{kind: kindConn, label: c.Name, conn: c})
	}
	if len(s.tables) > 0 {
		items = append(items, sidebarItem{kind: kindHeader, label: "TABLES"})
		for _, t := range s.tables {
			label := t.Name
			if t.Schema != "public" {
				label = t.Schema + "." + t.Name
			}
			items = append(items, sidebarItem{kind: kindTable, label: label, table: t})
		}
	}
	s.items = items
	s.clampCursor()
}

// clampCursor moves the cursor onto the nearest selectable row.
func (s *Sidebar) clampCursor() {
	if len(s.items) == 0 {
		s.cursor = 0
		return
	}
	if s.cursor >= len(s.items) {
		s.cursor = len(s.items) - 1
	}
	if s.cursor < 0 {
		s.cursor = 0
	}
	if !s.items[s.cursor].selectable() {
		s.moveCursor(1)
		if !s.items[s.cursor].selectable() {
			s.moveCursor(-1)
		}
	}
}

// moveCursor advances the cursor by step, skipping headers, without wrapping.
func (s *Sidebar) moveCursor(step int) {
	i := s.cursor
	for {
		i += step
		if i < 0 || i >= len(s.items) {
			return // no selectable row in that direction; leave cursor put
		}
		if s.items[i].selectable() {
			s.cursor = i
			return
		}
	}
}

// Selected returns the item under the cursor, if any is selectable.
func (s *Sidebar) Selected() (sidebarItem, bool) {
	if s.cursor >= 0 && s.cursor < len(s.items) && s.items[s.cursor].selectable() {
		return s.items[s.cursor], true
	}
	return sidebarItem{}, false
}

var sidebarKeys = struct {
	Up, Down, Enter, New, Edit, Delete key.Binding
}{
	Up:     key.NewBinding(key.WithKeys("up", "k")),
	Down:   key.NewBinding(key.WithKeys("down", "j")),
	Enter:  key.NewBinding(key.WithKeys("enter")),
	New:    key.NewBinding(key.WithKeys("n")),
	Edit:   key.NewBinding(key.WithKeys("e")),
	Delete: key.NewBinding(key.WithKeys("d")),
}

func (s *Sidebar) Update(msg tea.Msg) tea.Cmd {
	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok || !s.focused {
		return nil
	}
	switch {
	case key.Matches(keyMsg, sidebarKeys.Up):
		s.moveCursor(-1)
	case key.Matches(keyMsg, sidebarKeys.Down):
		s.moveCursor(1)
	case key.Matches(keyMsg, sidebarKeys.New):
		return func() tea.Msg { return messages.NewConnectionMsg{} }
	case key.Matches(keyMsg, sidebarKeys.Enter):
		if it, ok := s.Selected(); ok {
			switch it.kind {
			case kindConn:
				c := it.conn
				return func() tea.Msg { return messages.ConnectRequestMsg{Conn: c} }
			case kindTable:
				t := it.table
				return func() tea.Msg { return messages.RunTableMsg{Table: t} }
			}
		}
	case key.Matches(keyMsg, sidebarKeys.Edit):
		if it, ok := s.Selected(); ok && it.kind == kindConn {
			c := it.conn
			return func() tea.Msg { return messages.EditConnectionMsg{Conn: c} }
		}
	case key.Matches(keyMsg, sidebarKeys.Delete):
		if it, ok := s.Selected(); ok && it.kind == kindConn {
			name := it.conn.Name
			return func() tea.Msg { return messages.DeleteConnectionMsg{Name: name} }
		}
	}
	return nil
}

func (s *Sidebar) View() string {
	inner := theme.InnerWidth(s.width)
	var b strings.Builder
	for i, it := range s.items {
		line := s.renderItem(i, it, inner)
		b.WriteString(line)
		if i < len(s.items)-1 {
			b.WriteByte('\n')
		}
	}
	body := b.String()
	// Clip to the available height.
	body = clipHeight(body, theme.InnerHeight(s.height))
	return theme.Box("Database", s.focused, s.width, s.height, body)
}

func (s *Sidebar) renderItem(i int, it sidebarItem, width int) string {
	switch it.kind {
	case kindHeader:
		return lipgloss.NewStyle().Foreground(theme.Muted).Bold(true).Render(it.label)
	default:
		label := it.label
		prefix := "  "
		if it.kind == kindConn && it.conn.Name == s.activeConn {
			prefix = "● "
		} else if it.kind == kindTable {
			prefix = "  "
		}
		text := truncate(prefix+label, width)
		style := lipgloss.NewStyle()
		if i == s.cursor && s.focused {
			style = style.Foreground(lipgloss.Color("0")).Background(theme.Accent)
		} else if i == s.cursor {
			style = style.Foreground(theme.Accent)
		}
		return style.Render(padRight(text, width))
	}
}
