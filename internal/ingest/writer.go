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

// InsertEvalRows inserts flattened evaluation rows.
// ReplacingMergeTree deduplicates on row_key during merges.
func (w *Writer) InsertEvalRows(ctx context.Context, rows []EvalRow) error {
	if len(rows) == 0 {
		return nil
	}

	batch, err := w.conn.PrepareBatch(ctx, `INSERT INTO evaluation_logs (
		log_id, target_id, target_env, policy_id, catalog_ref_id,
		control_id, control_name, control_result,
		requirement_id, plan_id, assessment_result,
		message, description, applicability,
		steps_executed, confidence_level, recommendation,
		collected_at, completed_at
	)`)
	if err != nil {
		return fmt.Errorf("prepare batch: %w", err)
	}

	for _, r := range rows {
		if err := batch.Append(
			r.LogID, r.TargetID, r.TargetEnv, r.PolicyID, r.CatalogRefID,
			r.ControlID, r.ControlName, r.ControlResult,
			r.RequirementID, r.PlanID, r.AssessmentResult,
			r.Message, r.Description, r.Applicability,
			r.StepsExecuted, r.ConfidenceLevel, r.Recommendation,
			r.CollectedAt, r.CompletedAt,
		); err != nil {
			return fmt.Errorf("append eval row: %w", err)
		}
	}
	return batch.Send()
}

// InsertEnforcementRows inserts flattened enforcement action rows.
// ReplacingMergeTree deduplicates on row_key during merges.
func (w *Writer) InsertEnforcementRows(ctx context.Context, rows []EnforcementRow) error {
	if len(rows) == 0 {
		return nil
	}

	batch, err := w.conn.PrepareBatch(ctx, `INSERT INTO enforcement_actions (
		log_id, target_id, target_env, policy_id, catalog_ref_id,
		control_id, requirement_id,
		disposition, method_id, assessment_result, eval_log_ref,
		message, has_exception, exception_refs,
		started_at, completed_at
	)`)
	if err != nil {
		return fmt.Errorf("prepare batch: %w", err)
	}

	for _, r := range rows {
		if err := batch.Append(
			r.LogID, r.TargetID, r.TargetEnv, r.PolicyID, r.CatalogRefID,
			r.ControlID, r.RequirementID,
			r.Disposition, r.MethodID, r.AssessmentResult, r.EvalLogRef,
			r.Message, r.HasException, r.ExceptionRefs,
			r.StartedAt, r.CompletedAt,
		); err != nil {
			return fmt.Errorf("append enforcement row: %w", err)
		}
	}
	return batch.Send()
}
