package config

import (
	"strings"
	"testing"
)

func TestRedactConnString(t *testing.T) {
	cases := map[string]string{
		"postgres://user:secret@host:5432/db": "postgres://user:***@host:5432/db",
		"postgres://user@host/db":             "postgres://user@host/db",
		"host=localhost dbname=app":           "host=localhost dbname=app",
	}
	for in, want := range cases {
		if got := redactConnString(in); got != want {
			t.Errorf("redactConnString(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestSummaryNeverLeaksPassword(t *testing.T) {
	c := Connection{Name: "x", Method: MethodConnString, ConnString: "postgres://u:hunter2@h/db"}
	if strings.Contains(c.Summary(), "hunter2") {
		t.Errorf("summary leaked password: %q", c.Summary())
	}
}

func TestStoreAddReplacesByName(t *testing.T) {
	var s Store
	s.Add(Connection{Name: "a", ConnString: "one"})
	s.Add(Connection{Name: "a", ConnString: "two"})
	if len(s.Connections) != 1 {
		t.Fatalf("expected 1 connection, got %d", len(s.Connections))
	}
	if s.Connections[0].ConnString != "two" {
		t.Errorf("expected replacement, got %q", s.Connections[0].ConnString)
	}
}

func TestStoreRemove(t *testing.T) {
	s := Store{Connections: []Connection{{Name: "a"}, {Name: "b"}}}
	s.Remove("a")
	if len(s.Connections) != 1 || s.Connections[0].Name != "b" {
		t.Errorf("unexpected connections after remove: %+v", s.Connections)
	}
}
