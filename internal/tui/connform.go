package tui

import (
	"strings"

	"github.com/aurelio-labs/quaere/internal/config"
	"github.com/aurelio-labs/quaere/internal/tui/theme"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// formResult is what a ConnForm.Update reports back to the root model.
type formResult int

const (
	formNone formResult = iota
	formSaved
	formCancelled
)

// ConnForm is the modal for creating or editing a saved connection. Row 0 is the
// method selector; the remaining rows are text inputs that depend on the method.
type ConnForm struct {
	editing  bool
	origName string // original name when editing (so a rename replaces cleanly)

	method   config.Method
	name     textinput.Model
	connStr  textinput.Model
	instance textinput.Model
	user     textinput.Model
	database textinput.Model

	focus int // 0 = method row, then 1..n over the visible inputs
	err   string

	width, height int
}

// newConnForm returns a blank form for creating a connection.
func newConnForm() *ConnForm {
	f := &ConnForm{method: config.MethodConnString}
	f.name = newInput("my database", 60)
	f.connStr = newInput("postgres://user:pass@host:5432/dbname", 200)
	f.instance = newInput("project:region:instance", 120)
	f.user = newInput("me@example.com", 120)
	f.database = newInput("postgres", 80)
	f.focus = 0
	f.syncFocus()
	return f
}

// editConnForm returns a form pre-filled from an existing connection.
func editConnForm(c config.Connection) *ConnForm {
	f := newConnForm()
	f.editing = true
	f.origName = c.Name
	f.method = c.Method
	f.name.SetValue(c.Name)
	f.connStr.SetValue(c.ConnString)
	f.instance.SetValue(c.Instance)
	f.user.SetValue(c.User)
	f.database.SetValue(c.Database)
	f.syncFocus()
	return f
}

func newInput(placeholder string, limit int) textinput.Model {
	ti := textinput.New()
	ti.Placeholder = placeholder
	ti.CharLimit = limit
	ti.Width = 60
	return ti
}

func (f *ConnForm) SetSize(w, h int) { f.width, f.height = w, h }

// inputs returns the text inputs visible for the current method, in row order
// (the name field is always first).
func (f *ConnForm) inputs() []*textinput.Model {
	switch f.method {
	case config.MethodCloudSQLIAM:
		return []*textinput.Model{&f.name, &f.instance, &f.user, &f.database}
	default:
		return []*textinput.Model{&f.name, &f.connStr}
	}
}

// rowCount is the number of focusable rows (method row + inputs).
func (f *ConnForm) rowCount() int { return 1 + len(f.inputs()) }

// syncFocus focuses exactly the input matching f.focus (none if on the method row).
func (f *ConnForm) syncFocus() {
	for i, in := range f.inputs() {
		if f.focus == i+1 {
			in.Focus()
		} else {
			in.Blur()
		}
	}
	if f.focus >= f.rowCount() {
		f.focus = f.rowCount() - 1
	}
}

func (f *ConnForm) Update(msg tea.Msg) (formResult, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return formNone, nil
	}
	switch keyMsg.String() {
	case "esc":
		return formCancelled, nil
	case "ctrl+s":
		return f.trySave()
	case "tab", "down":
		f.focus = (f.focus + 1) % f.rowCount()
		f.syncFocus()
		return formNone, nil
	case "shift+tab", "up":
		f.focus--
		if f.focus < 0 {
			f.focus = f.rowCount() - 1
		}
		f.syncFocus()
		return formNone, nil
	case "enter":
		// Enter advances through the fields, and saves from the last row.
		if f.focus == f.rowCount()-1 {
			return f.trySave()
		}
		f.focus++
		f.syncFocus()
		return formNone, nil
	case "left", "right":
		if f.focus == 0 { // toggle method on the selector row
			if f.method == config.MethodConnString {
				f.method = config.MethodCloudSQLIAM
			} else {
				f.method = config.MethodConnString
			}
			f.syncFocus()
			return formNone, nil
		}
	}

	// Route typing to the focused input.
	if f.focus > 0 {
		inputs := f.inputs()
		idx := f.focus - 1
		if idx < len(inputs) {
			var cmd tea.Cmd
			*inputs[idx], cmd = inputs[idx].Update(msg)
			return formNone, cmd
		}
	}
	return formNone, nil
}

// trySave validates the form and, if valid, reports formSaved.
func (f *ConnForm) trySave() (formResult, tea.Cmd) {
	if strings.TrimSpace(f.name.Value()) == "" {
		f.err = "name is required"
		return formNone, nil
	}
	switch f.method {
	case config.MethodConnString:
		if strings.TrimSpace(f.connStr.Value()) == "" {
			f.err = "connection string is required"
			return formNone, nil
		}
	case config.MethodCloudSQLIAM:
		if strings.TrimSpace(f.instance.Value()) == "" {
			f.err = "instance connection name is required"
			return formNone, nil
		}
		if strings.TrimSpace(f.user.Value()) == "" {
			f.err = "IAM user is required"
			return formNone, nil
		}
		if strings.TrimSpace(f.database.Value()) == "" {
			f.err = "database is required"
			return formNone, nil
		}
	}
	f.err = ""
	return formSaved, nil
}

// Connection builds a config.Connection from the current field values.
func (f *ConnForm) Connection() config.Connection {
	c := config.Connection{
		Name:   strings.TrimSpace(f.name.Value()),
		Method: f.method,
	}
	switch f.method {
	case config.MethodConnString:
		c.ConnString = strings.TrimSpace(f.connStr.Value())
	case config.MethodCloudSQLIAM:
		c.Instance = strings.TrimSpace(f.instance.Value())
		c.User = strings.TrimSpace(f.user.Value())
		c.Database = strings.TrimSpace(f.database.Value())
	}
	return c
}

func (f *ConnForm) View() string {
	title := "New connection"
	if f.editing {
		title = "Edit connection"
	}

	var b strings.Builder
	b.WriteString(theme.TitleStyle.Foreground(theme.Primary).Render(title))
	b.WriteString("\n\n")

	// Method selector row.
	b.WriteString(f.methodRow())
	b.WriteString("\n\n")

	// Input rows.
	b.WriteString(f.inputRow("Name", &f.name, 1))
	switch f.method {
	case config.MethodConnString:
		b.WriteString(f.inputRow("Connection string", &f.connStr, 2))
	case config.MethodCloudSQLIAM:
		b.WriteString(f.inputRow("Instance (project:region:instance)", &f.instance, 2))
		b.WriteString(f.inputRow("IAM user (email)", &f.user, 3))
		b.WriteString(f.inputRow("Database", &f.database, 4))
	}

	if f.err != "" {
		b.WriteString("\n")
		b.WriteString(lipgloss.NewStyle().Foreground(theme.Red).Render("⚠ " + f.err))
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(theme.MutedStyle.Render("tab/↑↓ move · ←→ toggle method · enter next/save · ctrl+s save · esc cancel"))

	inner := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(theme.Primary).
		Padding(1, 2).
		Render(b.String())

	// Center the modal in the available area.
	return lipgloss.Place(f.width, f.height, lipgloss.Center, lipgloss.Center, inner)
}

func (f *ConnForm) methodRow() string {
	sel := func(label string, active, focused bool) string {
		st := lipgloss.NewStyle().Padding(0, 2)
		switch {
		case active && focused:
			st = st.Foreground(lipgloss.Color("0")).Background(theme.Primary).Bold(true)
		case active:
			st = st.Foreground(theme.Primary).Bold(true)
		default:
			st = st.Foreground(theme.Muted)
		}
		return st.Render(label)
	}
	onRow := f.focus == 0
	label := theme.MutedStyle.Render("Method  ")
	cs := sel("Connection string", f.method == config.MethodConnString, onRow)
	gcp := sel("Cloud SQL (IAM)", f.method == config.MethodCloudSQLIAM, onRow)
	return label + cs + " " + gcp
}

func (f *ConnForm) inputRow(label string, in *textinput.Model, row int) string {
	lbl := theme.MutedStyle.Width(34).Render(label)
	if f.focus == row {
		lbl = lipgloss.NewStyle().Foreground(theme.Primary).Bold(true).Width(34).Render(label)
	}
	return lbl + "\n" + in.View() + "\n\n"
}
