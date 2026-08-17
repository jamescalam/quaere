package components

import (
	"github.com/aurelio-labs/quaere/internal/tui/theme"
	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
)

// QueryEditor is the SQL editor pane (a multi-line textarea). It never runs the
// query itself — the root model reads Value() when the global run key is pressed.
type QueryEditor struct {
	base
	area textarea.Model
}

func NewQueryEditor() *QueryEditor {
	ta := textarea.New()
	ta.Placeholder = "Write SQL here — ctrl+r to run"
	ta.ShowLineNumbers = true
	ta.CharLimit = 0 // unlimited
	ta.Blur()
	return &QueryEditor{area: ta}
}

func (e *QueryEditor) Init() tea.Cmd { return textarea.Blink }

// Value returns the current SQL text.
func (e *QueryEditor) Value() string { return e.area.Value() }

// SetValue replaces the editor contents (e.g. a generated SELECT for a table).
func (e *QueryEditor) SetValue(s string) { e.area.SetValue(s) }

func (e *QueryEditor) Focus() {
	e.base.Focus()
	e.area.Focus()
}

func (e *QueryEditor) Blur() {
	e.base.Blur()
	e.area.Blur()
}

func (e *QueryEditor) SetSize(w, h int) {
	e.base.SetSize(w, h)
	e.area.SetWidth(theme.InnerWidth(w))
	e.area.SetHeight(theme.InnerHeight(h))
}

func (e *QueryEditor) Update(msg tea.Msg) tea.Cmd {
	if !e.focused {
		return nil
	}
	var cmd tea.Cmd
	e.area, cmd = e.area.Update(msg)
	return cmd
}

func (e *QueryEditor) View() string {
	return theme.Box("Query", e.focused, e.width, e.height, e.area.View())
}
