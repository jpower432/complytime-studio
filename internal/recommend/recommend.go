// SPDX-License-Identifier: Apache-2.0

package recommend

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"slices"
	"sort"
	"strconv"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/complytime/complytime-studio/internal/postgres"
)

const (
	strengthScale    = 100.0
	evidenceNorm     = 10.0
	topRecommendN    = 10
	weightMapping    = 0.6
	weightEvidence   = 0.3
	weightAppOverlap = 0.1
)

// Recommendation is a scored policy suggestion for a program.
type Recommendation struct {
	PolicyID          string  `json:"policy_id"`
	PolicyTitle       string  `json:"policy_title"`
	Reason            string  `json:"reason"`
	MappingStrength   float64 `json:"mapping_strength"`
	EvidenceCount     int     `json:"evidence_count"`
	Score             float64 `json:"score"`
	PredictedScorePct *int    `json:"predicted_score_pct,omitempty"`
	ScoreDelta        *int    `json:"score_delta,omitempty"`
}

// Engine scores candidate policies from catalog crosswalks and evidence.
type Engine struct {
	pool *pgxpool.Pool
}

// New builds an engine that uses pool for reads and writes.
func New(pool *pgxpool.Pool) *Engine {
	return &Engine{pool: pool}
}

// ForProgram returns up to topRecommendN policy suggestions, sorted by composite score descending.
func (e *Engine) ForProgram(ctx context.Context, programID string) ([]Recommendation, error) {
	var guidance *string
	var policyIDs []string
	var applicability []string
	var currentScorePct int
	err := e.pool.QueryRow(ctx, `
		SELECT guidance_catalog_id, policy_ids, applicability, score_pct
		FROM programs
		WHERE id = $1::uuid AND deleted_at IS NULL`, programID,
	).Scan(&guidance, &policyIDs, &applicability, &currentScorePct)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("for program %s: %w", programID, postgres.ErrProgramNotFound)
		}
		return nil, fmt.Errorf("for program load: %w", err)
	}
	if guidance == nil || *guidance == "" {
		return nil, nil
	}
	if policyIDs == nil {
		policyIDs = []string{}
	}
	gid := *guidance

	rows, err := e.pool.Query(ctx, `
		SELECT ctrl.policy_id, MAX(me.strength)::double precision
		FROM mapping_entries me
		INNER JOIN controls ctrl ON (
			(me.source_catalog_id = $1 AND ctrl.catalog_id = me.target_catalog_id)
			OR (me.target_catalog_id = $1 AND ctrl.catalog_id = me.source_catalog_id)
		)
		INNER JOIN policies p ON p.policy_id = ctrl.policy_id
		WHERE ctrl.policy_id <> ''
		  AND NOT (ctrl.policy_id = ANY($2::text[]))
		  AND NOT EXISTS (
			SELECT 1 FROM recommendation_dismissals d
			WHERE d.program_id = $3::uuid AND d.policy_id = ctrl.policy_id
		  )
		GROUP BY ctrl.policy_id`, gid, policyIDs, programID)
	if err != nil {
		return nil, fmt.Errorf("for program candidates: %w", err)
	}
	defer rows.Close()

	type cand struct {
		policyID     string
		peakStrength float64
	}
	var candidates []cand
	for rows.Next() {
		var pid string
		var peak float64
		if err := rows.Scan(&pid, &peak); err != nil {
			return nil, fmt.Errorf("for program scan candidates: %w", err)
		}
		candidates = append(candidates, cand{policyID: pid, peakStrength: peak})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("for program candidates: %w", err)
	}
	if len(candidates) == 0 {
		return nil, nil
	}

	policyList := make([]string, len(candidates))
	for i, c := range candidates {
		policyList[i] = c.policyID
	}

	evidenceCounts, tagSets, err := e.evidenceAgg(ctx, policyList)
	if err != nil {
		return nil, err
	}

	titles, err := e.policyTitles(ctx, policyList)
	if err != nil {
		return nil, err
	}

	out := make([]Recommendation, 0, len(candidates))
	for _, c := range candidates {
		mapStrength := math.Min(math.Max(c.peakStrength/strengthScale, 0), 1)
		evCount := evidenceCounts[c.policyID]
		evFactor := math.Min(float64(evCount)/evidenceNorm, 1)
		appOv := applicabilityOverlap(applicability, tagSets[c.policyID])
		score := weightMapping*mapStrength + weightEvidence*evFactor + weightAppOverlap*appOv

		title := titles[c.policyID]
		if title == "" {
			title = c.policyID
		}
		reason := buildReason(mapStrength, title, evCount)

		out = append(out, Recommendation{
			PolicyID:        c.policyID,
			PolicyTitle:     title,
			Reason:          reason,
			MappingStrength: mapStrength,
			EvidenceCount:   evCount,
			Score:           score,
		})
	}

	sort.Slice(out, func(i, j int) bool { return out[i].Score > out[j].Score })
	if len(out) > topRecommendN {
		out = out[:topRecommendN]
	}

	e.enrichWithPredictedPosture(ctx, out, policyIDs, currentScorePct)
	return out, nil
}

// enrichWithPredictedPosture simulates attaching each recommended policy and
// computes what the posture score would become. Best-effort: failures are
// silently skipped (predicted fields remain nil).
func (e *Engine) enrichWithPredictedPosture(ctx context.Context, recs []Recommendation, currentPolicyIDs []string, currentScorePct int) {
	for i := range recs {
		hypothetical := append(slices.Clone(currentPolicyIDs), recs[i].PolicyID)
		var passN, failN, errN int
		err := e.pool.QueryRow(ctx, `
			WITH latest AS (
				SELECT DISTINCT ON (target_id, policy_id, evidence_id, control_id, requirement_id)
					eval_result
				FROM evidence
				WHERE policy_id = ANY($1::text[])
				ORDER BY target_id, policy_id, evidence_id, control_id, requirement_id, collected_at DESC
			)
			SELECT
				COUNT(*) FILTER (WHERE eval_result = 'Passed'),
				COUNT(*) FILTER (WHERE eval_result = 'Failed'),
				COUNT(*) FILTER (WHERE eval_result = 'Needs Review')
			FROM latest`, hypothetical).Scan(&passN, &failN, &errN)
		if err != nil {
			slog.Debug("predicted posture query failed", "policy_id", recs[i].PolicyID, "error", err)
			continue
		}
		den := passN + failN + errN
		if den == 0 {
			continue
		}
		predicted := int(math.Round(float64(passN) / float64(den) * 100))
		delta := predicted - currentScorePct
		recs[i].PredictedScorePct = &predicted
		recs[i].ScoreDelta = &delta
	}
}

func (e *Engine) evidenceAgg(ctx context.Context, policyIDs []string) (counts map[string]int, tags map[string]map[string]struct{}, err error) {
	counts = make(map[string]int)
	tags = make(map[string]map[string]struct{})
	rows, err := e.pool.Query(ctx, `
		SELECT policy_id, control_applicability
		FROM evidence
		WHERE policy_id = ANY($1::text[])`, policyIDs)
	if err != nil {
		return nil, nil, fmt.Errorf("evidence aggregate: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var pid string
		var app []string
		if err := rows.Scan(&pid, &app); err != nil {
			return nil, nil, fmt.Errorf("evidence aggregate scan: %w", err)
		}
		counts[pid]++
		if tags[pid] == nil {
			tags[pid] = make(map[string]struct{})
		}
		for _, a := range app {
			if a != "" {
				tags[pid][a] = struct{}{}
			}
		}
	}
	if err := rows.Err(); err != nil {
		return nil, nil, fmt.Errorf("evidence aggregate: %w", err)
	}
	return counts, tags, nil
}

func (e *Engine) policyTitles(ctx context.Context, policyIDs []string) (map[string]string, error) {
	rows, err := e.pool.Query(ctx, `
		SELECT policy_id, title FROM policies WHERE policy_id = ANY($1::text[])`, policyIDs)
	if err != nil {
		return nil, fmt.Errorf("policy titles: %w", err)
	}
	defer rows.Close()
	out := make(map[string]string)
	for rows.Next() {
		var id, title string
		if err := rows.Scan(&id, &title); err != nil {
			return nil, fmt.Errorf("policy titles scan: %w", err)
		}
		out[id] = title
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("policy titles: %w", err)
	}
	return out, nil
}

func applicabilityOverlap(programApps []string, evidenceTags map[string]struct{}) float64 {
	if len(programApps) == 0 || len(evidenceTags) == 0 {
		return 0
	}
	matches := 0
	for _, a := range programApps {
		if a == "" {
			continue
		}
		if _, ok := evidenceTags[a]; ok {
			matches++
		}
	}
	return float64(matches) / float64(len(programApps))
}

func buildReason(mapStrength float64, policyTitle string, evCount int) string {
	var label string
	switch {
	case mapStrength >= 0.7:
		label = "Strong"
	case mapStrength >= 0.4:
		label = "Moderate"
	default:
		label = "Light"
	}
	pct := int(math.Round(mapStrength * 100))
	return label + " mapping (" + strconv.Itoa(pct) + "%) to " + policyTitle + " with " + strconv.Itoa(evCount) + " evidence items"
}

// Dismiss records that the user chose to hide a recommendation for the program.
func (e *Engine) Dismiss(ctx context.Context, programID, policyID, userID string) error {
	_, err := e.pool.Exec(ctx, `
		INSERT INTO recommendation_dismissals (program_id, policy_id, user_id)
		VALUES ($1::uuid, $2, $3)
		ON CONFLICT (program_id, policy_id) DO UPDATE
		SET user_id = EXCLUDED.user_id, dismissed_at = now()`, programID, policyID, userID)
	if err != nil {
		return fmt.Errorf("dismiss recommendation: %w", err)
	}
	return nil
}

// Undismiss removes a dismissal so the policy may appear in recommendations again.
func (e *Engine) Undismiss(ctx context.Context, programID, policyID string) error {
	_, err := e.pool.Exec(ctx, `
		DELETE FROM recommendation_dismissals
		WHERE program_id = $1::uuid AND policy_id = $2`, programID, policyID)
	if err != nil {
		return fmt.Errorf("undismiss recommendation: %w", err)
	}
	return nil
}
