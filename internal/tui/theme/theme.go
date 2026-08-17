// Package theme holds quaere's shared colors and a bordered-box renderer. It is
// a leaf package (imported by both tui and its components) so pane chrome stays
// consistent without an import cycle.
package theme

import "github.com/charmbracelet/lipgloss"

// Palette. Blue is the "focused / active" accent; the rest are supporting tones.
const (
	Primary = lipgloss.Color("75")  // active pane border, focus accent
	Accent  = lipgloss.Color("135") // headers, selected row
	Green   = lipgloss.Color("42")  // success / connected
	Red     = lipgloss.Color("203") // errors
	Muted   = lipgloss.Color("245") // secondary text, help
	Border  = lipgloss.Color("240") // idle pane border
)

var (
	// TitleStyle renders a pane title.
	TitleStyle = lipgloss.NewStyle().Bold(true)
	// MutedStyle renders secondary/help text.
	MutedStyle = lipgloss.NewStyle().Foreground(Muted)
)

// InnerWidth is the usable content width inside a Box of the given outer width
// (subtracting the two border columns).
func InnerWidth(w int) int { return max(0, w-2) }

// InnerHeight is the usable content height inside a Box of the given outer
// height (subtracting the two border rows and the one title row).
func InnerHeight(h int) int { return max(0, h-3) }

// Box renders body inside a rounded border sized to width×height, with a title
// on the first interior line. The border is highlighted when focused. Body is
// expected to already fit InnerWidth(width)×InnerHeight(height).
func Box(title string, focused bool, width, height int, body string) string {
	bc := Border
	tc := Muted
	if focused {
		bc = Primary
		tc = Primary
	}
	title = TitleStyle.Foreground(tc).Render(title)
	content := lipgloss.JoinVertical(lipgloss.Left, title, body)
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(bc).
		Width(InnerWidth(width)).
		Height(InnerHeight(height) + 1). // +1 for the title line
		Render(content)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
