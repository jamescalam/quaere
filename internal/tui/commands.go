package tui

import (
	"context"
	"time"

	"github.com/aurelio-labs/quaere/internal/config"
	"github.com/aurelio-labs/quaere/internal/db"
	"github.com/aurelio-labs/quaere/internal/gcp"
	tea "github.com/charmbracelet/bubbletea"
)

const (
	connectTimeout = 15 * time.Second
	queryTimeout   = 60 * time.Second
)

// connectedMsg is emitted when a connection opens successfully. warn is set when
// the connection is live but table introspection failed (e.g. limited perms).
type connectedMsg struct {
	conn   config.Connection
	client *db.Client
	tables []db.Table
	warn   error
}

// connectErrMsg is emitted when a connection could not be opened.
type connectErrMsg struct {
	conn config.Connection
	err  error
}

// queryResultMsg / queryErrMsg carry the outcome of a run.
type (
	queryResultMsg struct{ rs *db.ResultSet }
	queryErrMsg    struct{ err error }
)

// gcpLoginDoneMsg is emitted after the interactive gcloud login finishes; conn
// is the connection we were trying to open so the flow can resume.
type gcpLoginDoneMsg struct {
	conn config.Connection
	err  error
}

// connectCmd opens conn and, on success, loads its table list.
func connectCmd(conn config.Connection) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), connectTimeout)
		defer cancel()

		client, err := db.Connect(ctx, conn)
		if err != nil {
			return connectErrMsg{conn: conn, err: err}
		}
		tables, terr := client.Tables(ctx)
		return connectedMsg{conn: conn, client: client, tables: tables, warn: terr}
	}
}

// runQueryCmd executes sql against client.
func runQueryCmd(client *db.Client, sql string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), queryTimeout)
		defer cancel()

		rs, err := client.Run(ctx, sql)
		if err != nil {
			return queryErrMsg{err: err}
		}
		return queryResultMsg{rs: rs}
	}
}

// gcpLoginCmd runs the interactive `gcloud auth application-default login`,
// yielding the terminal to it, then reports completion so the connect can retry.
func gcpLoginCmd(conn config.Connection) tea.Cmd {
	return tea.ExecProcess(gcp.LoginCmd(), func(err error) tea.Msg {
		return gcpLoginDoneMsg{conn: conn, err: err}
	})
}
