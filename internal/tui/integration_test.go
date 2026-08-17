package tui

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/aurelio-labs/quaere/internal/config"
	"github.com/aurelio-labs/quaere/internal/tui/messages"
	tea "github.com/charmbracelet/bubbletea"
)

// TestConnectAndQueryLive drives the full model against a real PostgreSQL,
// exercising Connect → introspection → query → render exactly as the running TUI
// would. Set QUAERE_PG_DSN to enable it, e.g.
//
//	QUAERE_PG_DSN='postgres://postgres:secret@localhost:55432/app?sslmode=disable' go test ./internal/tui -run Live
func TestConnectAndQueryLive(t *testing.T) {
	dsn := os.Getenv("QUAERE_PG_DSN")
	if dsn == "" {
		t.Skip("set QUAERE_PG_DSN to run the live database integration test")
	}

	conn := config.Connection{Name: "live", Method: config.MethodConnString, ConnString: dsn}
	m := New(&config.Store{Connections: []config.Connection{conn}})
	m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})

	// Connect (runs the async connect command, then feeds its result back in).
	msg := drain(t, m, messages.ConnectRequestMsg{Conn: conn})
	cm, ok := msg.(connectedMsg)
	if !ok {
		t.Fatalf("expected connectedMsg, got %T (%v)", msg, msg)
	}
	if cm.warn != nil {
		t.Fatalf("introspection failed: %v", cm.warn)
	}
	m.Update(cm)
	defer m.closeClient()

	if v := m.View(); !strings.Contains(v, "users") || !strings.Contains(v, "TABLES") {
		t.Errorf("sidebar missing introspected tables:\n%s", v)
	}

	// Run a query.
	msg = drain(t, m, messages.RunQueryMsg{SQL: "SELECT name, email FROM users ORDER BY id"})
	qr, ok := msg.(queryResultMsg)
	if !ok {
		t.Fatalf("expected queryResultMsg, got %T (%v)", msg, msg)
	}
	if len(qr.rs.Rows) != 3 {
		t.Fatalf("expected 3 rows, got %d", len(qr.rs.Rows))
	}
	m.Update(qr)

	if v := m.View(); !strings.Contains(v, "Ada") || !strings.Contains(v, "ada@x.com") {
		t.Errorf("results grid missing expected data:\n%s", v)
	}
}

// drain sends msg to the model and runs the returned command (if any) to get the
// resulting async message, with a timeout guard.
func drain(t *testing.T, m *Model, msg tea.Msg) tea.Msg {
	t.Helper()
	_, cmd := m.Update(msg)
	if cmd == nil {
		t.Fatalf("expected a command from %T", msg)
	}
	done := make(chan tea.Msg, 1)
	go func() { done <- cmd() }()
	select {
	case out := <-done:
		return out
	case <-time.After(20 * time.Second):
		t.Fatal("command timed out")
		return nil
	}
}
