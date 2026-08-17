package components

import (
	"github.com/aurelio-labs/quaere/internal/tui/theme"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// StatusBar is the single-line bar above the help line. It shows the active
// connection on the left and a transient status/error message on the right.
type StatusBar struct {
	base
	conn    string
	message string
	isError bool
}

func NewStatusBar() *StatusBar { return &StatusBar{conn: "not connected"} }

func (s *StatusBar) Init() tea.Cmd          { return nil }
func (s *StatusBar) Update(tea.Msg) tea.Cmd { return nil }

// SetConnection sets the active-connection label (left side).
func (s *StatusBar) SetConnection(label string) {
	if label == "" {
		label = "not connected"
	}
	s.conn = label
}

// SetMessage sets a transient status message (right side).
func (s *StatusBar) SetMessage(msg string, isError bool) {
	s.message = msg
	s.isError = isError
}

func (s *StatusBar) View() string {
	left := lipgloss.NewStyle().Foreground(theme.Green).Bold(true).Render(" " + s.conn + " ")

	msgStyle := lipgloss.NewStyle().Foreground(theme.Muted)
	if s.isError {
		msgStyle = lipgloss.NewStyle().Foreground(theme.Red)
	}
	right := msgStyle.Render(s.message + " ")

	gap := s.width - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 0 {
		gap = 0
	}
	bar := left + lipgloss.NewStyle().Width(gap).Render("") + right
	return lipgloss.NewStyle().
		Width(s.width).
		Background(lipgloss.Color("236")).
		Render(bar)
}
