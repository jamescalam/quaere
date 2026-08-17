package cmd

import (
	"fmt"
	"os"

	"github.com/aurelio-labs/quaere/internal/config"
	"github.com/aurelio-labs/quaere/internal/tui"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
)

// version is overridden at build time via -ldflags "-X ...cmd.version=...".
var version = "dev"

var rootCmd = &cobra.Command{
	Use:   "quaere",
	Short: "A terminal SQL client (PostgreSQL + Cloud SQL IAM)",
	Long: `quaere is a terminal-based SQL client inspired by Beekeeper Studio.

Saved connections and their tables live in a left sidebar; the main area holds
a query editor on top and a results grid below. Connect either with a direct
PostgreSQL connection string or a Google Cloud SQL instance via IAM auth.`,
	Version:       version,
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runTUI()
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func runTUI() error {
	store, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to read saved connections: %w", err)
	}

	model := tui.New(store)
	program := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := program.Run(); err != nil {
		return fmt.Errorf("failed to run TUI: %w", err)
	}
	return nil
}
