// Package config resolves quaere's on-disk locations and persists the list of
// saved connections between runs.
package config

import (
	"os"
	"path/filepath"
)

func xdg(envVar string, fallback ...string) string {
	if v := os.Getenv(envVar); v != "" {
		return v
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(append([]string{home}, fallback...)...)
}

// ConfigDir is where quaere's connection store lives (small, backed up).
func ConfigDir() string { return filepath.Join(xdg("XDG_CONFIG_HOME", ".config"), "quaere") }

// storePath is the JSON file holding saved connections.
func storePath() string { return filepath.Join(ConfigDir(), "connections.json") }
