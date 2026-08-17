// Package messages holds the tea.Msg types passed between quaere's components
// and its root model.
package messages

import (
	"github.com/aurelio-labs/quaere/internal/config"
	"github.com/aurelio-labs/quaere/internal/db"
)

// ConnectRequestMsg asks the root model to open a saved connection (emitted when
// the user presses enter on a connection in the sidebar).
type ConnectRequestMsg struct{ Conn config.Connection }

// RunTableMsg asks the root to load a table: it fills the editor with a SELECT
// and runs it (emitted when the user presses enter on a table in the sidebar).
type RunTableMsg struct{ Table db.Table }

// RunQueryMsg asks the root to execute the given SQL against the active client.
type RunQueryMsg struct{ SQL string }

// NewConnectionMsg asks the root to open the new-connection form.
type NewConnectionMsg struct{}

// EditConnectionMsg asks the root to open the form pre-filled for editing.
type EditConnectionMsg struct{ Conn config.Connection }

// DeleteConnectionMsg asks the root to remove a saved connection by name.
type DeleteConnectionMsg struct{ Name string }
