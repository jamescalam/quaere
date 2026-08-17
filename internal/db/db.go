// Package db is quaere's PostgreSQL access layer. Both connection methods —
// a direct connection string and a Cloud SQL IAM instance — resolve to a single
// pgxpool-backed client, so query execution and schema introspection are shared.
package db

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Column is a single column of a table, used to populate the sidebar tree.
type Column struct {
	Name string
	Type string
}

// Table is one relation (table or view) discovered by introspection.
type Table struct {
	Schema string
	Name   string
	Type   string // "BASE TABLE", "VIEW", ...
}

// Qualified returns the schema-qualified, quoted identifier for use in SQL.
func (t Table) Qualified() string {
	return quoteIdent(t.Schema) + "." + quoteIdent(t.Name)
}

// ResultSet is the outcome of running a statement: the returned rows (empty for
// non-SELECT statements) plus the server's command tag and wall-clock time.
type ResultSet struct {
	Columns []string
	Rows    [][]string
	Command string // e.g. "SELECT 100" or "UPDATE 3"
	Elapsed time.Duration
}

// Client is a live connection to a PostgreSQL database.
type Client struct {
	pool   *pgxpool.Pool
	closer func() // tears down any auxiliary resources (e.g. the Cloud SQL dialer)
}

// Close releases the pool and any auxiliary resources.
func (c *Client) Close() {
	if c == nil {
		return
	}
	if c.pool != nil {
		c.pool.Close()
	}
	if c.closer != nil {
		c.closer()
	}
}

// Run executes an arbitrary statement and returns its result. SELECTs yield
// rows; other statements yield an empty row set and a command tag describing the
// effect. The whole statement text is sent as-is (a single query per call).
func (c *Client) Run(ctx context.Context, sql string) (*ResultSet, error) {
	start := time.Now()
	rows, err := c.pool.Query(ctx, sql)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	fds := rows.FieldDescriptions()
	cols := make([]string, len(fds))
	for i, fd := range fds {
		cols[i] = fd.Name
	}

	var data [][]string
	for rows.Next() {
		vals, err := rows.Values()
		if err != nil {
			return nil, err
		}
		row := make([]string, len(vals))
		for i, v := range vals {
			row[i] = formatValue(v)
		}
		data = append(data, row)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return &ResultSet{
		Columns: cols,
		Rows:    data,
		Command: rows.CommandTag().String(),
		Elapsed: time.Since(start),
	}, nil
}

// Tables lists user tables and views across all non-system schemas.
func (c *Client) Tables(ctx context.Context) ([]Table, error) {
	const q = `
		SELECT table_schema, table_name, table_type
		FROM information_schema.tables
		WHERE table_schema NOT IN ('pg_catalog', 'information_schema')
		ORDER BY table_schema, table_name`
	rows, err := c.pool.Query(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Table
	for rows.Next() {
		var t Table
		if err := rows.Scan(&t.Schema, &t.Name, &t.Type); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// Columns lists the columns of one table, in ordinal order.
func (c *Client) Columns(ctx context.Context, schema, table string) ([]Column, error) {
	const q = `
		SELECT column_name, data_type
		FROM information_schema.columns
		WHERE table_schema = $1 AND table_name = $2
		ORDER BY ordinal_position`
	rows, err := c.pool.Query(ctx, q, schema, table)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Column
	for rows.Next() {
		var col Column
		if err := rows.Scan(&col.Name, &col.Type); err != nil {
			return nil, err
		}
		out = append(out, col)
	}
	return out, rows.Err()
}

// formatValue renders a scanned value as a display string. NULLs become the
// literal "NULL"; byte slices are shown as text; times use RFC3339.
func formatValue(v any) string {
	switch x := v.(type) {
	case nil:
		return "NULL"
	case []byte:
		return string(x)
	case string:
		return x
	case time.Time:
		return x.Format(time.RFC3339)
	default:
		return fmt.Sprintf("%v", x)
	}
}

// quoteIdent double-quotes a SQL identifier, escaping any embedded quotes.
func quoteIdent(s string) string {
	return `"` + strings.ReplaceAll(s, `"`, `""`) + `"`
}
