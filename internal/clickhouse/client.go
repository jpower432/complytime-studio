// SPDX-License-Identifier: Apache-2.0

package clickhouse

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

// Client wraps a ClickHouse connection with health checking and query methods.
type Client struct {
	conn driver.Conn
}

// Config holds ClickHouse connection parameters.
type Config struct {
	Addr     string
	Database string
	User     string
	Password string
}

// New creates a ClickHouse client and verifies connectivity.
func New(ctx context.Context, cfg Config) (*Client, error) {
	if cfg.Database == "" {
		cfg.Database = "default"
	}
	conn, err := clickhouse.Open(&clickhouse.Options{
		Addr: []string{cfg.Addr},
		Auth: clickhouse.Auth{
			Database: cfg.Database,
			Username: cfg.User,
			Password: cfg.Password,
		},
		DialTimeout:     5 * time.Second,
		ConnMaxLifetime: 10 * time.Minute,
	})
	if err != nil {
		return nil, fmt.Errorf("clickhouse open: %w", err)
	}
	if err := conn.Ping(ctx); err != nil {
		return nil, fmt.Errorf("clickhouse ping: %w", err)
	}
	slog.Info("clickhouse connected", "addr", cfg.Addr, "database", cfg.Database)
	return &Client{conn: conn}, nil
}

// Ping checks connectivity.
func (c *Client) Ping(ctx context.Context) error {
	return c.conn.Ping(ctx)
}

// Conn returns the underlying driver connection for direct queries.
func (c *Client) Conn() driver.Conn {
	return c.conn
}

// EnsureSchema creates required tables if they don't exist.
// Safe to call on every startup — all statements use IF NOT EXISTS.
func (c *Client) EnsureSchema(ctx context.Context, retentionMonths int) error {
	if retentionMonths <= 0 {
		retentionMonths = 24
	}

	stmts := []string{
		fmt.Sprintf(`CREATE TABLE IF NOT EXISTS evidence (
			evidence_id String,
			target_id String,
			target_name Nullable(String),
			target_type Nullable(String),
			target_env Nullable(String),
			engine_name Nullable(String),
			engine_version Nullable(String),
			rule_id String,
			rule_name Nullable(String),
			rule_uri Nullable(String),
			eval_result Enum8('Not Run'=0,'Passed'=1,'Failed'=2,'Needs Review'=3,'Not Applicable'=4,'Unknown'=5),
			eval_message Nullable(String),
			policy_id String DEFAULT '',
			control_id String DEFAULT '',
			control_catalog_id Nullable(String),
			control_category Nullable(String),
			control_applicability Array(String),
			requirement_id String DEFAULT '',
			plan_id Nullable(String),
			confidence Nullable(Enum8('Undetermined'=0,'Low'=1,'Medium'=2,'High'=3)),
			steps_executed Nullable(UInt16),
			compliance_status Enum8('Compliant'=0,'Non-Compliant'=1,'Exempt'=2,'Not Applicable'=3,'Unknown'=4),
			risk_level Nullable(Enum8('Critical'=0,'High'=1,'Medium'=2,'Low'=3,'Informational'=4)),
			frameworks Array(String),
			requirements Array(String),
			remediation_action Nullable(Enum8('Block'=0,'Allow'=1,'Remediate'=2,'Waive'=3,'Notify'=4,'Unknown'=5)),
			remediation_status Nullable(Enum8('Success'=0,'Fail'=1,'Skipped'=2,'Unknown'=3)),
			remediation_desc Nullable(String),
			exception_id Nullable(String),
			exception_active Nullable(Bool),
			enrichment_status Enum8('Success'=0,'Unmapped'=1,'Partial'=2,'Unknown'=3,'Skipped'=4),
			collected_at DateTime64(3),
			ingested_at DateTime64(3) DEFAULT now64(3),
			row_key String MATERIALIZED concat(evidence_id,'/',control_id,'/',requirement_id)
		) ENGINE = ReplacingMergeTree(ingested_at)
		PARTITION BY toYYYYMM(collected_at)
		ORDER BY (target_id, policy_id, control_id, collected_at, row_key)
		TTL toDateTime(collected_at) + INTERVAL %d MONTH`, retentionMonths),

		`CREATE TABLE IF NOT EXISTS policies (
			policy_id String,
			title String,
			version Nullable(String),
			oci_reference String,
			content String,
			imported_at DateTime64(3) DEFAULT now64(3),
			imported_by Nullable(String)
		) ENGINE = ReplacingMergeTree(imported_at)
		ORDER BY (policy_id)`,

		`CREATE TABLE IF NOT EXISTS mapping_documents (
			mapping_id String,
			policy_id String,
			framework String,
			content String,
			imported_at DateTime64(3) DEFAULT now64(3)
		) ENGINE = ReplacingMergeTree(imported_at)
		ORDER BY (mapping_id, policy_id)`,

		fmt.Sprintf(`CREATE TABLE IF NOT EXISTS audit_logs (
			audit_id String,
			policy_id String,
			audit_start DateTime64(3),
			audit_end DateTime64(3),
			framework Nullable(String),
			created_at DateTime64(3) DEFAULT now64(3),
			created_by Nullable(String),
			content String,
			summary String
		) ENGINE = ReplacingMergeTree(created_at)
		PARTITION BY toYYYYMM(audit_start)
		ORDER BY (policy_id, audit_start, audit_id)
		TTL toDateTime(audit_start) + INTERVAL %d MONTH`, retentionMonths),
	}

	for _, stmt := range stmts {
		if err := c.conn.Exec(ctx, stmt); err != nil {
			return fmt.Errorf("ensure schema: %w", err)
		}
	}
	slog.Info("clickhouse schema verified")
	return nil
}
