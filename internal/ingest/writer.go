// SPDX-License-Identifier: Apache-2.0

package ingest

import (
	"context"
	"fmt"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

// WriterConfig holds ClickHouse connection parameters.
type WriterConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	Database string
}

// Writer inserts flattened rows into ClickHouse.
type Writer struct {
	conn driver.Conn
}

// NewWriter opens a ClickHouse connection.
func NewWriter(cfg WriterConfig) (*Writer, error) {
	conn, err := clickhouse.Open(&clickhouse.Options{
		Addr: []string{fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)},
		Auth: clickhouse.Auth{
			Database: cfg.Database,
			Username: cfg.User,
			Password: cfg.Password,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("open clickhouse: %w", err)
	}
	if err := conn.Ping(context.Background()); err != nil {
		return nil, fmt.Errorf("ping clickhouse: %w", err)
	}
	return &Writer{conn: conn}, nil
}

// Close releases the ClickHouse connection.
func (w *Writer) Close() error {
	return w.conn.Close()
}

// InsertEvidenceRows inserts flattened evidence rows into the unified table.
// ReplacingMergeTree deduplicates on row_key during merges.
func (w *Writer) InsertEvidenceRows(ctx context.Context, rows []EvidenceRow) error {
	if len(rows) == 0 {
		return nil
	}

	batch, err := w.conn.PrepareBatch(ctx, `INSERT INTO evidence (
		evidence_id,
		target_id, target_name, target_type, target_env,
		engine_name, engine_version, rule_id, rule_name, rule_uri,
		eval_result, eval_message,
		policy_id, control_id, control_catalog_id, control_category,
		control_applicability, requirement_id, plan_id,
		confidence, steps_executed, compliance_status,
		risk_level, frameworks, requirements,
		remediation_action, remediation_status, remediation_desc,
		exception_id, exception_active,
		enrichment_status,
		collected_at
	)`)
	if err != nil {
		return fmt.Errorf("prepare batch: %w", err)
	}

	for _, r := range rows {
		if err := batch.Append(
			r.EvidenceID,
			r.TargetID, r.TargetName, r.TargetType, r.TargetEnv,
			r.EngineName, r.EngineVersion, r.RuleID, r.RuleName, r.RuleURI,
			r.EvalResult, r.EvalMessage,
			r.PolicyID, r.ControlID, r.ControlCatalogID, r.ControlCategory,
			r.ControlApplicability, r.RequirementID, r.PlanID,
			r.Confidence, r.StepsExecuted, r.ComplianceStatus,
			r.RiskLevel, r.Frameworks, r.Requirements,
			r.RemediationAction, r.RemediationStatus, r.RemediationDesc,
			r.ExceptionID, r.ExceptionActive,
			r.EnrichmentStatus,
			r.CollectedAt,
		); err != nil {
			return fmt.Errorf("append evidence row: %w", err)
		}
	}
	return batch.Send()
}
