package tui

import (
	"github.com/aurelio-labs/quaere/internal/tui/components"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type pane int

const (
	paneSidebar pane = iota
	paneEditor
	paneResults
)

// sidebarWidth is the fixed width of the left column.
const sidebarWidth = 34

// PaneManager owns the three panes, their layout, and which one has focus.
type PaneManager struct {
	sidebar *components.Sidebar
	editor  *components.QueryEditor
	results *components.ResultGrid

	focused       pane
	width, height int
}

func NewPaneManager(sidebar *components.Sidebar, editor *components.QueryEditor, results *components.ResultGrid) *PaneManager {
	pm := &PaneManager{sidebar: sidebar, editor: editor, results: results, focused: paneSidebar}
	sidebar.Focus()
	return pm
}

func (pm *PaneManager) Init() tea.Cmd {
	return tea.Batch(pm.sidebar.Init(), pm.editor.Init(), pm.results.Init())
}

// CycleFocus moves focus to the next (or previous) pane.
func (pm *PaneManager) CycleFocus(reverse bool) {
	pm.focusedComponent().Blur()
	if reverse {
		pm.focused = (pm.focused + 2) % 3
	} else {
		pm.focused = (pm.focused + 1) % 3
	}
	pm.focusedComponent().Focus()
}

// Focus moves focus to a specific pane.
func (pm *PaneManager) Focus(p pane) {
	pm.focusedComponent().Blur()
	pm.focused = p
	pm.focusedComponent().Focus()
}

func (pm *PaneManager) Focused() pane { return pm.focused }

func (pm *PaneManager) focusedComponent() components.Component {
	switch pm.focused {
	case paneEditor:
		return pm.editor
	case paneResults:
		return pm.results
	default:
		return pm.sidebar
	}
}

// Update routes a message to the focused pane (used for key/scroll input).
func (pm *PaneManager) Update(msg tea.Msg) tea.Cmd {
	return pm.focusedComponent().Update(msg)
}

// SetSize recomputes the layout: full-height sidebar on the left; editor over
// results on the right.
func (pm *PaneManager) SetSize(w, h int) {
	pm.width, pm.height = w, h

	sw := sidebarWidth
	if sw > w/2 {
		sw = w / 2
	}
	rightW := w - sw

	editorH := h / 3
	if editorH < 6 {
		editorH = 6
	}
	if editorH > h-4 {
		editorH = max(1, h-4)
	}
	resultsH := h - editorH

	pm.sidebar.SetSize(sw, h)
	pm.editor.SetSize(rightW, editorH)
	pm.results.SetSize(rightW, resultsH)
}

func (pm *PaneManager) View() string {
	right := lipgloss.JoinVertical(lipgloss.Left, pm.editor.View(), pm.results.View())
	return lipgloss.JoinHorizontal(lipgloss.Top, pm.sidebar.View(), right)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
