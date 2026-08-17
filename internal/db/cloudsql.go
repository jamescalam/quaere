package db

import (
	"context"
	"fmt"
	"net"

	"cloud.google.com/go/cloudsqlconn"
	"github.com/aurelio-labs/quaere/internal/config"
	"github.com/jackc/pgx/v5/pgxpool"
)

// connectCloudSQLIAM opens a pool to a Cloud SQL for PostgreSQL instance using
// IAM database authentication. Credentials come from Application Default
// Credentials (the same ones `gcloud auth application-default login` writes), so
// no password is stored. The cloudsqlconn dialer terminates TLS to the instance,
// which is why the pgx DSN uses sslmode=disable — pgx must not attempt its own
// TLS on top of the already-secured tunnel.
func connectCloudSQLIAM(ctx context.Context, conn config.Connection) (*Client, error) {
	if conn.Instance == "" {
		return nil, fmt.Errorf("cloud sql instance connection name is empty")
	}
	if conn.User == "" {
		return nil, fmt.Errorf("cloud sql IAM user is empty")
	}
	if conn.Database == "" {
		return nil, fmt.Errorf("cloud sql database is empty")
	}

	dialer, err := cloudsqlconn.NewDialer(ctx, cloudsqlconn.WithIAMAuthN())
	if err != nil {
		return nil, fmt.Errorf("create cloud sql dialer: %w", err)
	}

	dsn := fmt.Sprintf("user=%s dbname=%s sslmode=disable", conn.User, conn.Database)
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		dialer.Close()
		return nil, err
	}
	instance := conn.Instance
	cfg.ConnConfig.DialFunc = func(ctx context.Context, _, _ string) (net.Conn, error) {
		return dialer.Dial(ctx, instance)
	}

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		dialer.Close()
		return nil, err
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		dialer.Close()
		return nil, err
	}

	return &Client{pool: pool, closer: func() { dialer.Close() }}, nil
}
