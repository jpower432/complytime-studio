// SPDX-License-Identifier: Apache-2.0

package ingest

import "time"

// EvidenceRow is a flattened row for the unified `evidence` ClickHouse table.
// Co-locates evaluation and remediation data; remediation fields are nil
// for evaluation-only records.
type EvidenceRow struct {
	EvidenceID string `ch:"evidence_id"`

	TargetID   string  `ch:"target_id"`
	TargetName *string `ch:"target_name"`
	TargetType *string `ch:"target_type"`
	TargetEnv  *string `ch:"target_env"`

	EngineName    *string `ch:"engine_name"`
	EngineVersion *string `ch:"engine_version"`
	RuleID        string  `ch:"rule_id"`
	RuleName      *string `ch:"rule_name"`
	RuleURI       *string `ch:"rule_uri"`

	EvalResult  string  `ch:"eval_result"`
	EvalMessage *string `ch:"eval_message"`

	PolicyID             *string  `ch:"policy_id"`
	ControlID            *string  `ch:"control_id"`
	ControlCatalogID     *string  `ch:"control_catalog_id"`
	ControlCategory      *string  `ch:"control_category"`
	ControlApplicability []string `ch:"control_applicability"`
	RequirementID        *string  `ch:"requirement_id"`
	PlanID               *string  `ch:"plan_id"`
	Confidence           *string  `ch:"confidence"`
	StepsExecuted        *uint16  `ch:"steps_executed"`
	ComplianceStatus     string   `ch:"compliance_status"`
	RiskLevel            *string  `ch:"risk_level"`
	Frameworks           []string `ch:"frameworks"`
	Requirements         []string `ch:"requirements"`

	RemediationAction *string `ch:"remediation_action"`
	RemediationStatus *string `ch:"remediation_status"`
	RemediationDesc   *string `ch:"remediation_desc"`
	ExceptionID       *string `ch:"exception_id"`
	ExceptionActive   *bool   `ch:"exception_active"`

	EnrichmentStatus string `ch:"enrichment_status"`

	CollectedAt time.Time `ch:"collected_at"`
}

func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func uint16Ptr(v uint16) *uint16 {
	if v == 0 {
		return nil
	}
	return &v
}
