package tui

import (
	"strings"
	"testing"

	"github.com/aurelio-labs/quaere/internal/config"
	"github.com/aurelio-labs/quaere/internal/tui/messages"
	tea "github.com/charmbracelet/bubbletea"
)

func testModel() *Model {
	store := &config.Store{Connections: []config.Connection{
		{Name: "local", Method: config.MethodConnString, ConnString: "postgres://localhost/app"},
		{Name: "prod", Method: config.MethodCloudSQLIAM, Instance: "p:r:i", User: "me@x.com", Database: "app"},
	}}
	m := New(store)
	m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	return m
}

func TestViewRendersPanes(t *testing.T) {
	m := testModel()
	view := m.View()
	for _, want := range []string{"CONNECTIONS", "local", "prod", "Query", "Results"} {
		if !strings.Contains(view, want) {
			t.Errorf("view missing %q\n%s", want, view)
		}
	}
}

func TestNewConnectionOpensForm(t *testing.T) {
	m := testModel()
	m.Update(messages.NewConnectionMsg{})
	if m.form == nil {
		t.Fatal("expected form to open")
	}
	if got := m.View(); !strings.Contains(got, "New connection") {
		t.Errorf("form view missing title\n%s", got)
	}
}

func TestTabCyclesFocus(t *testing.T) {
	m := testModel()
	if m.pm.Focused() != paneSidebar {
		t.Fatalf("expected initial focus on sidebar, got %v", m.pm.Focused())
	}
	m.Update(tea.KeyMsg{Type: tea.KeyTab})
	if m.pm.Focused() != paneEditor {
		t.Errorf("expected editor focus after tab, got %v", m.pm.Focused())
	}
}

func TestRunQueryWithoutClientWarns(t *testing.T) {
	m := testModel()
	// Nothing should panic and the status should flag the missing connection.
	m.Update(messages.RunQueryMsg{SQL: "SELECT 1"})
	if !strings.Contains(m.View(), "not connected") {
		t.Errorf("expected 'not connected' warning in view\n%s", m.View())
	}
}
