// SPDX-License-Identifier: Apache-2.0

package posture

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"sort"
	"time"

	"github.com/complytime/complytime-studio/internal/postgres"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Summary struct {
	ProgramID     string          `json:"program_id"`
	TotalPolicies int             `json:"total_policies"`
	PassCount     int             `json:"pass_count"`
	FailCount     int             `json:"fail_count"`
	ErrorCount    int             `json:"error_count"`
	UnknownCount  int             `json:"unknown_count"`
	ScorePct      int             `json:"score_pct"`
	Health        string          `json:"health"`
	Targets       []TargetSummary `json:"targets"`
	ComputedAt    time.Time       `json:"computed_at"`
}

type TargetSummary struct {
	TargetID  string `json:"target_id"`
	PolicyID  string `json:"policy_id"`
	PassCount int    `json:"pass_count"`
	FailCount int    `json:"fail_count"`
	Total     int    `json:"total"`
	Result    string `json:"result"`
}

type Engine struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Engine {
	return &Engine{pool: pool}
}

const latestEvidencePerGrain = `
WITH latest AS (
	SELECT DISTINCT ON (target_id, policy_id, evidence_id, control_id, requirement_id)
		target_id,
		policy_id,
		eval_result
	FROM evidence
	WHERE policy_id = ANY($1::text[])
	ORDER BY target_id, policy_id, evidence_id, control_id, requirement_id, collected_at DESC
)
SELECT
	target_id,
	policy_id,
	COUNT(*) FILTER (WHERE eval_result = 'Passed') AS pass_count,
	COUNT(*) FILTER (WHERE eval_result = 'Failed') AS fail_count,
	COUNT(*) FILTER (WHERE eval_result = 'Needs Review') AS error_count,
	COUNT(*) FILTER (WHERE eval_result IN ('Unknown', 'Not Run', 'Not Applicable')) AS unknown_count
FROM latest
GROUP BY target_id, policy_id
ORDER BY target_id, policy_id`

func (e *Engine) Compute(
	ctx context.Context, programID string, policyIDs []string, greenPct, redPct int,
) (*Summary, error) {
	if e == nil || e.pool == nil {
		return nil, errors.New("posture: nil engine or pool")
	}
	summary := &Summary{
		ProgramID:     programID,
		TotalPolicies: countPolicies(policyIDs),
		Targets:       nil,
		ComputedAt:    time.Now().UTC(),
	}
	if summary.TotalPolicies == 0 {
		summary.Health = classifyHealth(0, greenPct, redPct)
		return summary, nil
	}

	rows, err := e.pool.Query(ctx, latestEvidencePerGrain, policyIDs)
	if err != nil {
		return nil, fmt.Errorf("posture compute query: %w", err)
	}
	defer rows.Close()

	var targets []TargetSummary
	for rows.Next() {
		var (
			targetID, policyID       string
			passN, failN, errN, unkN int
		)
		if err := rows.Scan(&targetID, &policyID, &passN, &failN, &errN, &unkN); err != nil {
			return nil, fmt.Errorf("posture compute scan: %w", err)
		}
		summary.PassCount += passN
		summary.FailCount += failN
		summary.ErrorCount += errN
		summary.UnknownCount += unkN
		targets = append(targets, TargetSummary{
			TargetID:  targetID,
			PolicyID:  policyID,
			PassCount: passN,
			FailCount: failN,
			Total:     passN + failN + errN + unkN,
			Result:    rollupTargetResult(passN, failN, errN, unkN),
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("posture compute rows: %w", err)
	}

	sort.Slice(targets, func(i, j int) bool {
		if targets[i].TargetID != targets[j].TargetID {
			return targets[i].TargetID < targets[j].TargetID
		}
		return targets[i].PolicyID < targets[j].PolicyID
	})
	summary.Targets = targets

	den := summary.PassCount + summary.FailCount + summary.ErrorCount
	if den > 0 {
		summary.ScorePct = int(math.Round(float64(summary.PassCount) / float64(den) * 100))
	}
	summary.Health = classifyHealth(summary.ScorePct, greenPct, redPct)
	return summary, nil
}

func (e *Engine) ComputeAndStore(
	ctx context.Context, programID string, policyIDs []string, greenPct, redPct int,
) (*Summary, error) {
	summary, err := e.Compute(ctx, programID, policyIDs, greenPct, redPct)
	if err != nil {
		return nil, err
	}
	var version int
	err = e.pool.QueryRow(ctx, `
		SELECT version FROM programs WHERE id = $1 AND deleted_at IS NULL`,
		programID,
	).Scan(&version)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("update program health: %w", postgres.ErrProgramNotFound)
		}
		return nil, fmt.Errorf("update program health read version: %w", err)
	}
	tag, err := e.pool.Exec(ctx, `
		UPDATE programs
		SET health = $1, score_pct = $4, version = version + 1, updated_at = now()
		WHERE id = $2 AND version = $3 AND deleted_at IS NULL`,
		summary.Health, programID, version, summary.ScorePct,
	)
	if err != nil {
		return nil, fmt.Errorf("update program health: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return nil, fmt.Errorf("update program health: %w", postgres.ErrProgramVersionConflict)
	}
	return summary, nil
}

// RecomputePosture delegates to ComputeAndStore, discarding the summary.
// Callers may invoke asynchronously; async lifecycle is a caller concern.
func (e *Engine) RecomputePosture(ctx context.Context, programID string, policyIDs []string, greenPct, redPct int) error {
	_, err := e.ComputeAndStore(ctx, programID, policyIDs, greenPct, redPct)
	return err
}

func countPolicies(policyIDs []string) int {
	n := 0
	for _, id := range policyIDs {
		if id != "" {
			n++
		}
	}
	return n
}

func rollupTargetResult(pass, fail, errCnt, unk int) string {
	switch {
	case fail > 0:
		return "fail"
	case errCnt > 0:
		return "error"
	case unk > 0:
		return "unknown"
	case pass > 0:
		return "pass"
	default:
		return "unknown"
	}
}

func classifyHealth(scorePct, greenPct, redPct int) string {
	if scorePct >= greenPct {
		return "green"
	}
	if scorePct <= redPct {
		return "red"
	}
	return "yellow"
}

// PopulatePosture recomputes posture for all programs that have policies
// attached. Safe to call on every startup — uses direct UPDATE (no
// optimistic lock) to avoid version conflicts during boot.
func (e *Engine) PopulatePosture(ctx context.Context) error {
	if e == nil || e.pool == nil {
		return nil
	}
	rows, err := e.pool.Query(ctx, `
		SELECT id, policy_ids, green_pct, red_pct
		FROM programs
		WHERE deleted_at IS NULL AND array_length(policy_ids, 1) > 0`)
	if err != nil {
		return fmt.Errorf("populate posture list: %w", err)
	}
	defer rows.Close()

	type prog struct {
		id        string
		policyIDs []string
		greenPct  int
		redPct    int
	}
	var programs []prog
	for rows.Next() {
		var p prog
		if err := rows.Scan(&p.id, &p.policyIDs, &p.greenPct, &p.redPct); err != nil {
			return fmt.Errorf("populate posture scan: %w", err)
		}
		programs = append(programs, p)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("populate posture rows: %w", err)
	}

	var updated, failed int
	for _, p := range programs {
		summary, err := e.Compute(ctx, p.id, p.policyIDs, p.greenPct, p.redPct)
		if err != nil {
			slog.Warn("posture backfill compute failed", "program_id", p.id, "error", err)
			failed++
			continue
		}
		_, err = e.pool.Exec(ctx, `
			UPDATE programs
			SET health = $1, score_pct = $2, updated_at = now()
			WHERE id = $3 AND deleted_at IS NULL`,
			summary.Health, summary.ScorePct, p.id)
		if err != nil {
			slog.Warn("posture backfill update failed", "program_id", p.id, "error", err)
			failed++
			continue
		}
		updated++
	}
	if updated > 0 || failed > 0 {
		slog.Info("posture backfill complete", "updated", updated, "failed", failed, "total", len(programs))
	}
	return nil
}
