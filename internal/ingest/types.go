// SPDX-License-Identifier: Apache-2.0

package ingest

import "time"

// EvalRow is a flattened row for the evaluation_logs ClickHouse table.
// One row per AssessmentLog entry within a ControlEvaluation.
type EvalRow struct {
	LogID            string    `ch:"log_id"`
	TargetID         string    `ch:"target_id"`
	TargetEnv        string    `ch:"target_env"`
	PolicyID         string    `ch:"policy_id"`
	CatalogRefID     string    `ch:"catalog_ref_id"`
	ControlID        string    `ch:"control_id"`
	ControlName      string    `ch:"control_name"`
	ControlResult    string    `ch:"control_result"`
	RequirementID    string    `ch:"requirement_id"`
	PlanID           *string   `ch:"plan_id"`
	AssessmentResult string    `ch:"assessment_result"`
	Message          string    `ch:"message"`
	Description      string    `ch:"description"`
	Applicability    []string  `ch:"applicability"`
	StepsExecuted    uint16    `ch:"steps_executed"`
	ConfidenceLevel  string    `ch:"confidence_level"`
	Recommendation   *string   `ch:"recommendation"`
	CollectedAt      time.Time `ch:"collected_at"`
	CompletedAt      *time.Time `ch:"completed_at"`
}

// EnforcementRow is a flattened row for the enforcement_actions ClickHouse table.
// One row per AssessmentFinding within an ActionResult's justification.
type EnforcementRow struct {
	LogID            string    `ch:"log_id"`
	TargetID         string    `ch:"target_id"`
	TargetEnv        string    `ch:"target_env"`
	PolicyID         string    `ch:"policy_id"`
	CatalogRefID     string    `ch:"catalog_ref_id"`
	ControlID        string    `ch:"control_id"`
	RequirementID    string    `ch:"requirement_id"`
	Disposition      string    `ch:"disposition"`
	MethodID         string    `ch:"method_id"`
	AssessmentResult string    `ch:"assessment_result"`
	EvalLogRef       string    `ch:"eval_log_ref"`
	Message          *string   `ch:"message"`
	HasException     bool      `ch:"has_exception"`
	ExceptionRefs    []string  `ch:"exception_refs"`
	StartedAt        time.Time `ch:"started_at"`
	CompletedAt      *time.Time `ch:"completed_at"`
}
