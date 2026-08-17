package db

import (
	"context"
	"fmt"

	"github.com/aurelio-labs/quaere/internal/config"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Connect opens a pooled connection for a saved connection, dispatching on its
// method. The returned Client must be Closed by the caller.
func Connect(ctx context.Context, conn config.Connection) (*Client, error) {
	switch conn.Method {
	case config.MethodConnString:
		return connectConnString(ctx, conn)
	case config.MethodCloudSQLIAM:
		return connectCloudSQLIAM(ctx, conn)
	default:
		return nil, fmt.Errorf("unknown connection method %q", conn.Method)
	}
}

// connectConnString opens a pool straight from a PostgreSQL connection string.
func connectConnString(ctx context.Context, conn config.Connection) (*Client, error) {
	if conn.ConnString == "" {
		return nil, fmt.Errorf("connection string is empty")
	}
	pool, err := pgxpool.New(ctx, conn.ConnString)
	if err != nil {
		return nil, err
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	return &Client{pool: pool}, nil
}
