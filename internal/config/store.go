package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Method identifies how a connection authenticates and dials the database.
type Method string

const (
	// MethodConnString is a direct PostgreSQL connection string
	// (postgres://user:pass@host:port/db). The string may embed a password.
	MethodConnString Method = "conn_string"
	// MethodCloudSQLIAM connects to a Google Cloud SQL for PostgreSQL instance
	// using IAM database authentication over Application Default Credentials.
	// No password is stored: the IAM principal is the credential.
	MethodCloudSQLIAM Method = "cloudsql_iam"
)

// Connection is one saved database target shown in the sidebar.
type Connection struct {
	// Name is the human label shown in the sidebar.
	Name   string `json:"name"`
	Method Method `json:"method"`

	// ConnString method.
	ConnString string `json:"conn_string,omitempty"`

	// CloudSQLIAM method. Instance is the connection name
	// "project:region:instance"; User is the IAM database user (usually an
	// email or service-account address); Database is the target database.
	Instance string `json:"instance,omitempty"`
	User     string `json:"user,omitempty"`
	Database string `json:"database,omitempty"`

	LastUsed time.Time `json:"last_used,omitempty"`
}

// Summary is a short one-line description of where a connection points, safe to
// show in the UI (never includes a password).
func (c Connection) Summary() string {
	switch c.Method {
	case MethodCloudSQLIAM:
		return fmt.Sprintf("cloudsql · %s/%s", c.Instance, c.Database)
	case MethodConnString:
		return "postgres · " + redactConnString(c.ConnString)
	default:
		return string(c.Method)
	}
}

// redactConnString strips any password from a postgres URL for display.
func redactConnString(s string) string {
	// Handle URL form: postgres://user:pass@host/db -> postgres://user:***@host/db
	if at := strings.LastIndex(s, "@"); at != -1 {
		if scheme := strings.Index(s, "://"); scheme != -1 && scheme+3 < at {
			creds := s[scheme+3 : at]
			if colon := strings.Index(creds, ":"); colon != -1 {
				user := creds[:colon]
				return s[:scheme+3] + user + ":***@" + s[at+1:]
			}
		}
	}
	return s
}

// Store is the persisted set of saved connections.
type Store struct {
	Connections []Connection `json:"connections"`
}

// Add appends a connection, replacing any existing one with the same name.
func (s *Store) Add(c Connection) {
	for i, existing := range s.Connections {
		if existing.Name == c.Name {
			s.Connections[i] = c
			return
		}
	}
	s.Connections = append(s.Connections, c)
}

// Remove deletes the connection with the given name, if present.
func (s *Store) Remove(name string) {
	out := s.Connections[:0]
	for _, c := range s.Connections {
		if c.Name != name {
			out = append(out, c)
		}
	}
	s.Connections = out
}

// TouchLastUsed records that a connection was just opened and re-saves the
// store (best-effort; a save error is returned so callers may surface it).
func (s *Store) TouchLastUsed(name string) error {
	for i := range s.Connections {
		if s.Connections[i].Name == name {
			s.Connections[i].LastUsed = time.Now()
			break
		}
	}
	return Save(s)
}

// Load reads the on-disk connection store. A missing file is not an error — it
// returns an empty store so first-run users start with a clean slate.
func Load() (*Store, error) {
	var s Store
	data, err := os.ReadFile(storePath())
	if err != nil {
		if os.IsNotExist(err) {
			return &s, nil
		}
		return &s, err
	}
	if err := json.Unmarshal(data, &s); err != nil {
		return &s, err
	}
	return &s, nil
}

// Save writes the store to disk. Connection strings can embed passwords, so the
// directory and file are kept owner-only (0700 / 0600).
func Save(s *Store) error {
	path := storePath()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}
