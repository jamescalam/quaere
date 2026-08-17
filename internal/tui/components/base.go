// Package components holds quaere's focusable panes: the connection/table
// sidebar, the query editor, the results grid, and the status bar.
package components

import tea "github.com/charmbracelet/bubbletea"

// Component is the interface every pane implements. Update takes a message and
// returns any follow-up command; the pane mutates itself in place (pointer
// receiver).
type Component interface {
	Init() tea.Cmd
	Update(tea.Msg) tea.Cmd
	View() string
	SetSize(width, height int)
	Focus()
	Blur()
}

// base carries the geometry and focus state common to every component.
type base struct {
	width   int
	height  int
	focused bool
}

func (b *base) SetSize(w, h int) { b.width, b.height = w, h }
func (b *base) Focus()           { b.focused = true }
func (b *base) Blur()            { b.focused = false }
func (b *base) Focused() bool    { return b.focused }
