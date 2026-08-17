// Package tui is quaere's Bubble Tea layer: the root model, the pane manager,
// and the connection-form modal.
package tui

import (
	"context"
	"strconv"
	"strings"

	"github.com/aurelio-labs/quaere/internal/config"
	"github.com/aurelio-labs/quaere/internal/db"
	"github.com/aurelio-labs/quaere/internal/gcp"
	"github.com/aurelio-labs/quaere/internal/tui/components"
	"github.com/aurelio-labs/quaere/internal/tui/messages"
	"github.com/aurelio-labs/quaere/internal/tui/theme"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Model is quaere's root Bubble Tea model.
type Model struct {
	store *config.Store

	pm     *PaneManager
	status *components.StatusBar

	// form is non-nil while the connection modal is open.
	form *ConnForm

	client     *db.Client
	activeConn *config.Connection

	width, height int
}

// New builds the root model from the saved-connection store.
func New(store *config.Store) *Model {
	sidebar := components.NewSidebar(store.Connections)
	editor := components.NewQueryEditor()
	results := components.NewResultGrid()

	return &Model{
		store:  store,
		pm:     NewPaneManager(sidebar, editor, results),
		status: components.NewStatusBar(),
	}
}

func (m *Model) Init() tea.Cmd {
	return tea.Batch(m.pm.Init(), m.status.Init())
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.layout()
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)

	// ---- component requests ----
	case messages.NewConnectionMsg:
		m.form = newConnForm()
		m.form.SetSize(m.width, m.mainHeight())
		return m, nil

	case messages.EditConnectionMsg:
		m.form = editConnForm(msg.Conn)
		m.form.SetSize(m.width, m.mainHeight())
		return m, nil

	case messages.DeleteConnectionMsg:
		m.store.Remove(msg.Name)
		_ = config.Save(m.store)
		m.pm.sidebar.SetConnections(m.store.Connections)
		m.status.SetMessage("deleted "+msg.Name, false)
		return m, nil

	case messages.ConnectRequestMsg:
		return m, m.beginConnect(msg.Conn)

	case messages.RunTableMsg:
		return m, m.runTable(msg.Table)

	case messages.RunQueryMsg:
		return m, m.runQuery(msg.SQL)

	// ---- async results ----
	case connectedMsg:
		return m, m.onConnected(msg)

	case connectErrMsg:
		m.status.SetMessage("connect failed: "+msg.err.Error(), true)
		m.pm.results.SetError(msg.err)
		return m, nil

	case gcpLoginDoneMsg:
		if msg.err != nil {
			m.status.SetMessage("gcloud login failed: "+msg.err.Error(), true)
			return m, nil
		}
		m.status.SetMessage("login complete; connecting…", false)
		return m, connectCmd(msg.conn)

	case queryResultMsg:
		m.pm.results.SetResult(msg.rs)
		m.pm.Focus(paneResults)
		m.status.SetMessage("query ok", false)
		return m, nil

	case queryErrMsg:
		m.pm.results.SetError(msg.err)
		m.status.SetMessage("query error", true)
		return m, nil
	}

	// Forward anything else (e.g. cursor blink) to the focused pane.
	return m, m.pm.Update(msg)
}

// handleKey processes global keys, the modal, and otherwise routes to panes.
func (m *Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// The modal captures all keys while open.
	if m.form != nil {
		res, cmd := m.form.Update(msg)
		switch res {
		case formSaved:
			return m, m.onFormSaved()
		case formCancelled:
			m.form = nil
			m.status.SetMessage("cancelled", false)
		}
		return m, cmd
	}

	switch msg.String() {
	case "ctrl+c":
		m.closeClient()
		return m, tea.Quit
	case "q":
		// q quits unless the editor (which needs the letter) is focused.
		if m.pm.Focused() != paneEditor {
			m.closeClient()
			return m, tea.Quit
		}
	case "tab":
		m.pm.CycleFocus(false)
		return m, nil
	case "shift+tab":
		m.pm.CycleFocus(true)
		return m, nil
	case "ctrl+r":
		return m, m.runQuery(m.pm.editor.Value())
	}

	// Route to the focused pane (sidebar navigation, editor typing, grid scroll).
	return m, m.pm.Update(msg)
}

// onFormSaved persists the form's connection and closes the modal.
func (m *Model) onFormSaved() tea.Cmd {
	conn := m.form.Connection()
	// If editing renamed the connection, drop the old entry.
	if m.form.editing && m.form.origName != conn.Name {
		m.store.Remove(m.form.origName)
	}
	m.store.Add(conn)
	if err := config.Save(m.store); err != nil {
		m.status.SetMessage("save failed: "+err.Error(), true)
	} else {
		m.status.SetMessage("saved "+conn.Name, false)
	}
	m.pm.sidebar.SetConnections(m.store.Connections)
	m.form = nil
	return nil
}

// beginConnect opens a connection, first running an interactive gcloud login if
// the Cloud SQL IAM method has no Application Default Credentials yet.
func (m *Model) beginConnect(conn config.Connection) tea.Cmd {
	m.status.SetMessage("connecting to "+conn.Name+"…", false)
	m.pm.results.SetInfo("Connecting to " + conn.Name + "…")

	if conn.Method == config.MethodCloudSQLIAM && !gcp.HasADC(context.Background()) {
		if !gcp.GcloudInstalled() {
			m.status.SetMessage("gcloud CLI not found — install the Google Cloud SDK for IAM auth", true)
			return nil
		}
		m.status.SetMessage("opening browser for gcloud login…", false)
		return gcpLoginCmd(conn)
	}
	return connectCmd(conn)
}

// onConnected installs a freshly-opened client and its table list.
func (m *Model) onConnected(msg connectedMsg) tea.Cmd {
	m.closeClient()
	m.client = msg.client
	c := msg.conn
	m.activeConn = &c

	m.pm.sidebar.SetActive(c.Name)
	m.pm.sidebar.SetTables(msg.tables)
	m.status.SetConnection(c.Name + " · " + c.Summary())

	if msg.warn != nil {
		m.status.SetMessage("connected (tables unavailable: "+msg.warn.Error()+")", true)
	} else {
		m.status.SetMessage("connected · "+pluralTables(len(msg.tables)), false)
	}
	m.pm.results.SetInfo("Connected. Pick a table or write a query (ctrl+r to run).")

	_ = m.store.TouchLastUsed(c.Name)
	return nil
}

// runTable fills the editor with a preview SELECT and executes it.
func (m *Model) runTable(t db.Table) tea.Cmd {
	if m.client == nil {
		m.status.SetMessage("not connected", true)
		return nil
	}
	sql := "SELECT * FROM " + t.Qualified() + " LIMIT 100;"
	m.pm.editor.SetValue(sql)
	return m.runQuery(sql)
}

// runQuery executes sql against the active client.
func (m *Model) runQuery(sql string) tea.Cmd {
	if strings.TrimSpace(sql) == "" {
		return nil
	}
	if m.client == nil {
		m.status.SetMessage("not connected — select a connection first", true)
		return nil
	}
	m.status.SetMessage("running…", false)
	m.pm.results.SetInfo("Running…")
	m.pm.Focus(paneResults)
	return runQueryCmd(m.client, sql)
}

func (m *Model) closeClient() {
	if m.client != nil {
		m.client.Close()
		m.client = nil
	}
}

// layout sizes the panes, status bar, and any open modal.
func (m *Model) layout() {
	if m.width == 0 || m.height == 0 {
		return
	}
	m.pm.SetSize(m.width, m.mainHeight())
	m.status.SetSize(m.width, 1)
	if m.form != nil {
		m.form.SetSize(m.width, m.mainHeight())
	}
}

// mainHeight is the height available to the panes (reserving the status bar and
// help line).
func (m *Model) mainHeight() int { return max(1, m.height-2) }

func (m *Model) View() string {
	if m.width == 0 {
		return "Loading…"
	}

	var main string
	if m.form != nil {
		main = m.form.View()
	} else {
		main = m.pm.View()
	}

	return lipgloss.JoinVertical(lipgloss.Left, main, m.status.View(), m.helpView())
}

func (m *Model) helpView() string {
	var help string
	switch {
	case m.form != nil:
		help = "tab/↑↓ move · ←→ method · enter next/save · ctrl+s save · esc cancel"
	default:
		help = "tab switch pane · enter open/run · n new · e edit · d delete · ctrl+r run query · q quit"
	}
	return lipgloss.NewStyle().Foreground(theme.Muted).Padding(0, 1).Width(m.width).Render(help)
}

func pluralTables(n int) string {
	if n == 1 {
		return "1 table"
	}
	return strconv.Itoa(n) + " tables"
}
