// SPDX-License-Identifier: Apache-2.0

package postgres

import (
	"context"
	"embed"
	"fmt"
	"log/slog"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed migrations/*.sql
var migrationFS embed.FS

// Config holds PostgreSQL connection parameters.
type Config struct {
	URL string // Full connection string, e.g. postgres://user:pass@host:5432/studio
}

// ConfigFromEnv builds Config from the POSTGRES_URL env var.
// Returns ok=false when the variable is empty; the gateway treats this as fatal.
func ConfigFromEnv() (Config, bool) {
	url := os.Getenv("POSTGRES_URL")
	if url == "" {
		return Config{}, false
	}
	return Config{URL: url}, true
}

// Client wraps a pgxpool.Pool and provides schema migration.
type Client struct {
	pool *pgxpool.Pool
}

// New creates a connection pool and verifies connectivity.
func New(ctx context.Context, cfg Config) (*Client, error) {
	pool, err := pgxpool.New(ctx, cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("postgres connect: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("postgres ping: %w", err)
	}
	return &Client{pool: pool}, nil
}

// Pool returns the underlying connection pool for direct use by store implementations.
func (c *Client) Pool() *pgxpool.Pool {
	return c.pool
}

// Ping verifies the database connection is alive.
func (c *Client) Ping(ctx context.Context) error {
	return c.pool.Ping(ctx)
}

// Close releases the connection pool.
func (c *Client) Close() {
	c.pool.Close()
}

// EnsureSchema applies all embedded SQL migrations in order.
// Uses an advisory lock to prevent concurrent migration runs and a
// schema_migrations table to track applied versions.
func (c *Client) EnsureSchema(ctx context.Context) error {
	conn, err := c.pool.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("acquire conn for migration: %w", err)
	}
	defer conn.Release()

	// Advisory lock prevents concurrent migration runs across gateway replicas.
	const lockID int64 = 0x5354554449_4F5047 // "STUDIO_PG"
	if _, err := conn.Exec(ctx, "SELECT pg_advisory_lock($1)", lockID); err != nil {
		return fmt.Errorf("advisory lock: %w", err)
	}
	defer func() {
		_, _ = conn.Exec(ctx, "SELECT pg_advisory_unlock($1)", lockID)
	}()

	if _, err := conn.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version  INTEGER PRIMARY KEY,
			filename TEXT NOT NULL,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT now()
		)`); err != nil {
		return fmt.Errorf("create schema_migrations: %w", err)
	}

	entries, err := migrationFS.ReadDir("migrations")
	if err != nil {
		return fmt.Errorf("read migrations dir: %w", err)
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}
		parts := strings.SplitN(entry.Name(), "_", 2)
		version, err := strconv.Atoi(parts[0])
		if err != nil {
			return fmt.Errorf("migration %s: cannot parse version from filename: %w", entry.Name(), err)
		}

		var exists bool
		if err := conn.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version = $1)", version).Scan(&exists); err != nil {
			return fmt.Errorf("check migration %d: %w", version, err)
		}
		if exists {
			continue
		}

		sql, err := migrationFS.ReadFile("migrations/" + entry.Name())
		if err != nil {
			return fmt.Errorf("read migration %s: %w", entry.Name(), err)
		}

		slog.Info("applying postgres migration", "version", version, "file", entry.Name())
		if _, err := conn.Exec(ctx, string(sql)); err != nil {
			return fmt.Errorf("migration %d (%s): %w", version, entry.Name(), err)
		}

		if _, err := conn.Exec(ctx, "INSERT INTO schema_migrations (version, filename) VALUES ($1, $2)", version, entry.Name()); err != nil {
			return fmt.Errorf("record migration %d: %w", version, err)
		}
	}
	return nil
}
