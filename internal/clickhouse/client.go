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

// EnsureSchema creates required tables and applies incremental migrations.
// Safe to call on every startup — CREATE uses IF NOT EXISTS, migrations
// are tracked in schema_migrations and applied at most once.
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
			requirements Array(String),
			remediation_action Nullable(Enum8('Block'=0,'Allow'=1,'Remediate'=2,'Waive'=3,'Notify'=4,'Unknown'=5)),
			remediation_status Nullable(Enum8('Success'=0,'Fail'=1,'Skipped'=2,'Unknown'=3)),
			remediation_desc Nullable(String),
			exception_id Nullable(String),
			exception_active Nullable(Bool),
			enrichment_status Enum8('Success'=0,'Unmapped'=1,'Partial'=2,'Unknown'=3,'Skipped'=4),
			attestation_ref Nullable(String),
			source_registry Nullable(String),
			blob_ref Nullable(String),
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

		`CREATE TABLE IF NOT EXISTS mapping_entries (
			mapping_id String,
			policy_id String,
			control_id String,
			requirement_id String DEFAULT '',
			framework String,
			reference String,
			strength UInt8 DEFAULT 0,
			confidence String DEFAULT '',
			imported_at DateTime64(3) DEFAULT now64(3)
		) ENGINE = ReplacingMergeTree(imported_at)
		ORDER BY (policy_id, framework, control_id, reference)`,

		fmt.Sprintf(`CREATE TABLE IF NOT EXISTS audit_logs (
			audit_id String,
			policy_id String,
			audit_start DateTime64(3),
			audit_end DateTime64(3),
			framework Nullable(String),
			created_at DateTime64(3) DEFAULT now64(3),
			created_by Nullable(String),
			content String,
			summary String,
			model Nullable(String),
			prompt_version Nullable(String)
		) ENGINE = ReplacingMergeTree(created_at)
		PARTITION BY toYYYYMM(audit_start)
		ORDER BY (policy_id, audit_start, audit_id)
		TTL toDateTime(audit_start) + INTERVAL %d MONTH`, retentionMonths),

		`CREATE TABLE IF NOT EXISTS controls (
			catalog_id String,
			control_id String,
			title String,
			objective String,
			group_id String,
			state LowCardinality(String) DEFAULT 'Active',
			policy_id String DEFAULT '',
			imported_at DateTime64(3) DEFAULT now64(3)
		) ENGINE = ReplacingMergeTree(imported_at)
		ORDER BY (catalog_id, control_id)`,

		`CREATE TABLE IF NOT EXISTS assessment_requirements (
			catalog_id String,
			control_id String,
			requirement_id String,
			text String,
			applicability Array(String),
			recommendation String DEFAULT '',
			state LowCardinality(String) DEFAULT 'Active',
			imported_at DateTime64(3) DEFAULT now64(3)
		) ENGINE = ReplacingMergeTree(imported_at)
		ORDER BY (catalog_id, control_id, requirement_id)`,

		`CREATE TABLE IF NOT EXISTS control_threats (
			catalog_id String,
			control_id String,
			threat_reference_id String,
			threat_entry_id String,
			imported_at DateTime64(3) DEFAULT now64(3)
		) ENGINE = ReplacingMergeTree(imported_at)
		ORDER BY (catalog_id, control_id, threat_reference_id, threat_entry_id)`,

		`CREATE TABLE IF NOT EXISTS threats (
			catalog_id String,
			threat_id String,
			title String,
			description String,
			group_id String,
			policy_id String DEFAULT '',
			imported_at DateTime64(3) DEFAULT now64(3)
		) ENGINE = ReplacingMergeTree(imported_at)
		ORDER BY (catalog_id, threat_id)`,

		`CREATE TABLE IF NOT EXISTS risks (
			catalog_id String,
			risk_id String,
			title String,
			description String,
			severity LowCardinality(String) DEFAULT '',
			group_id String DEFAULT '',
			impact String DEFAULT '',
			policy_id String DEFAULT '',
			imported_at DateTime64(3) DEFAULT now64(3)
		) ENGINE = ReplacingMergeTree(imported_at)
		ORDER BY (catalog_id, risk_id)`,

		`CREATE TABLE IF NOT EXISTS risk_threats (
			catalog_id String,
			risk_id String,
			threat_reference_id String,
			threat_entry_id String,
			imported_at DateTime64(3) DEFAULT now64(3)
		) ENGINE = ReplacingMergeTree(imported_at)
		ORDER BY (catalog_id, risk_id, threat_reference_id, threat_entry_id)`,

		`CREATE TABLE IF NOT EXISTS catalogs (
			catalog_id String,
			catalog_type LowCardinality(String),
			title String,
			content String,
			policy_id String DEFAULT '',
			imported_at DateTime64(3) DEFAULT now64(3)
		) ENGINE = ReplacingMergeTree(imported_at)
		ORDER BY (catalog_id)`,
	}

	for _, stmt := range stmts {
		if err := c.conn.Exec(ctx, stmt); err != nil {
			return fmt.Errorf("ensure schema: %w", err)
		}
	}

	if err := c.applyMigrations(ctx); err != nil {
		return fmt.Errorf("apply migrations: %w", err)
	}

	slog.Info("clickhouse schema verified")
	return nil
}

type migration struct {
	Version     int
	Description string
	SQL         string
}

func schemaMigrations() []migration {
	return []migration{
		{
			Version:     1,
			Description: "add provenance columns to audit_logs",
			SQL:         `ALTER TABLE audit_logs ADD COLUMN IF NOT EXISTS model Nullable(String), ADD COLUMN IF NOT EXISTS prompt_version Nullable(String)`,
		},
		{
			Version:     2,
			Description: "drop denormalized frameworks from evidence",
			SQL:         `ALTER TABLE evidence DROP COLUMN IF EXISTS frameworks`,
		},
		{
			Version:     3,
			Description: "add attestation_ref to evidence",
			SQL:         `ALTER TABLE evidence ADD COLUMN IF NOT EXISTS attestation_ref Nullable(String)`,
		},
		{
			Version:     4,
			Description: "add noncompliant_evidence materialized view",
			SQL: `CREATE MATERIALIZED VIEW IF NOT EXISTS noncompliant_evidence
				ENGINE = ReplacingMergeTree(ingested_at)
				ORDER BY (policy_id, control_id, target_id, collected_at)
				AS SELECT * FROM evidence
				WHERE eval_result IN ('Failed', 'Needs Review')
				   OR compliance_status = 'Non-Compliant'`,
		},
		{
			Version:     5,
			Description: "add evidence_assessments table",
			SQL: `CREATE TABLE IF NOT EXISTS evidence_assessments (
				evidence_id String,
				policy_id String,
				plan_id String,
				classification Enum8('Healthy'=1,'Failing'=2,'Wrong Source'=3,'Wrong Method'=4,'Unfit Evidence'=5,'Stale'=6,'No Evidence'=7),
				reason String,
				assessed_at DateTime64(3),
				assessed_by String
			) ENGINE = MergeTree()
			PARTITION BY toYYYYMM(assessed_at)
			ORDER BY (policy_id, plan_id, evidence_id, assessed_at)`,
		},
		{
			Version:     6,
			Description: "add source_registry to evidence",
			SQL:         `ALTER TABLE evidence ADD COLUMN IF NOT EXISTS source_registry Nullable(String)`,
		},
		{
			Version:     7,
			Description: "add draft_audit_logs table",
			SQL: `CREATE TABLE IF NOT EXISTS draft_audit_logs (
				draft_id String,
				policy_id String,
				audit_start DateTime64(3),
				audit_end DateTime64(3),
				framework Nullable(String),
				created_at DateTime64(3) DEFAULT now64(3),
				status Enum8('pending_review'=1,'promoted'=2,'expired'=3),
				content String,
				summary String,
				agent_reasoning String DEFAULT '',
				model Nullable(String),
				prompt_version Nullable(String),
				reviewed_by Nullable(String),
				promoted_at Nullable(DateTime64(3)),
				reviewer_edits String DEFAULT '{}'
			) ENGINE = ReplacingMergeTree(created_at)
			PARTITION BY toYYYYMM(audit_start)
			ORDER BY (policy_id, audit_start, draft_id)
			TTL toDateTime(created_at) + INTERVAL 30 DAY`,
		},
		{
			Version:     8,
			Description: "add blob_ref to evidence",
			SQL:         `ALTER TABLE evidence ADD COLUMN IF NOT EXISTS blob_ref Nullable(String)`,
		},
		{
			Version:     9,
			Description: "add reviewer_edits to draft_audit_logs",
			SQL:         `ALTER TABLE draft_audit_logs ADD COLUMN IF NOT EXISTS reviewer_edits String DEFAULT '{}'`,
		},
		{
			Version:     10,
			Description: "add notifications table",
			SQL: `CREATE TABLE IF NOT EXISTS notifications (
				notification_id String,
				type LowCardinality(String),
				policy_id String,
				payload String DEFAULT '{}',
				read UInt8 DEFAULT 0,
				created_at DateTime64(3) DEFAULT now64(3)
			) ENGINE = ReplacingMergeTree(created_at)
			ORDER BY (notification_id)
			TTL toDateTime(created_at) + INTERVAL 90 DAY`,
		},
		{
			Version:     11,
			Description: "add certified column to evidence",
			SQL: `ALTER TABLE evidence
				ADD COLUMN IF NOT EXISTS certified Bool DEFAULT false`,
		},
		{
			Version:     12,
			Description: "add certifications table",
			SQL: `CREATE TABLE IF NOT EXISTS certifications (
				evidence_id       String,
				certifier         LowCardinality(String),
				certifier_version LowCardinality(String),
				result            Enum8(
				  'pass' = 1,
				  'fail' = 2,
				  'skip' = 3,
				  'error' = 4
				),
				reason            String,
				certified_at      DateTime64(3) DEFAULT now64(3)
			) ENGINE = MergeTree()
			PARTITION BY toYYYYMM(certified_at)
			ORDER BY (evidence_id, certifier, certified_at)`,
		},
		{
			Version:     13,
			Description: "add users table",
			SQL: `CREATE TABLE IF NOT EXISTS users (
				email String,
				name String,
				avatar_url String,
				role LowCardinality(String) DEFAULT 'reviewer',
				created_at DateTime64(3) DEFAULT now64(3),
				version UInt64 DEFAULT toUnixTimestamp64Milli(now64(3))
			) ENGINE = ReplacingMergeTree(version)
			ORDER BY (email)`,
		},
		{
			Version:     14,
			Description: "add role_changes audit table",
			SQL: `CREATE TABLE IF NOT EXISTS role_changes (
				changed_by String,
				target_email String,
				old_role LowCardinality(String),
				new_role LowCardinality(String),
				changed_at DateTime64(3) DEFAULT now64(3)
			) ENGINE = MergeTree
			ORDER BY (changed_at, target_email)`,
		},
		{
			Version:     15,
			Description: "add sub column to users table for generic OIDC support",
			// Single ADD COLUMN per statement — ClickHouse silently no-ops multi-column ADD COLUMN.
			SQL: `ALTER TABLE users ADD COLUMN IF NOT EXISTS sub String DEFAULT ''`,
		},
		{
			Version:     16,
			Description: "add issuer column to users table for generic OIDC support",
			SQL: `ALTER TABLE users ADD COLUMN IF NOT EXISTS issuer String DEFAULT ''`,
		},
	}
}

func (c *Client) applyMigrations(ctx context.Context) error {
	err := c.conn.Exec(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (
		version UInt32,
		description String,
		applied_at DateTime64(3) DEFAULT now64(3)
	) ENGINE = ReplacingMergeTree(applied_at)
	ORDER BY (version)`)
	if err != nil {
		return fmt.Errorf("create schema_migrations: %w", err)
	}

	applied, err := c.appliedVersions(ctx)
	if err != nil {
		return fmt.Errorf("read applied versions: %w", err)
	}

	for _, m := range schemaMigrations() {
		if applied[m.Version] {
			continue
		}
		slog.Info("applying migration", "version", m.Version, "description", m.Description)
		if err := c.conn.Exec(ctx, m.SQL); err != nil {
			return fmt.Errorf("migration %d (%s): %w", m.Version, m.Description, err)
		}
		if err := c.conn.Exec(ctx, `INSERT INTO schema_migrations (version, description) VALUES (?, ?)`, m.Version, m.Description); err != nil {
			return fmt.Errorf("record migration %d (%s): %w", m.Version, m.Description, err)
		}
	}
	return nil
}

func (c *Client) appliedVersions(ctx context.Context) (map[int]bool, error) {
	rows, err := c.conn.Query(ctx, `SELECT version FROM schema_migrations FINAL`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	applied := make(map[int]bool)
	for rows.Next() {
		var v uint32
		if err := rows.Scan(&v); err != nil {
			return nil, err
		}
		applied[int(v)] = true
	}
	return applied, rows.Err()
}
