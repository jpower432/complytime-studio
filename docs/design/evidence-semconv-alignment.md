# Evidence Semantic Convention Alignment

Mapping between the `beacon.evidence` OTel semantic convention ([complytime-collector-components/model](https://github.com/complytime/complytime-collector-components/tree/main/model)) and the Studio ClickHouse `evidence` table.

## Attribute-to-Column Mapping

### Policy Engine Attributes (`registry.policy`)

| Semconv Attribute | ClickHouse Column | Type | Requirement |
|:------------------|:------------------|:-----|:------------|
| `policy.engine.name` | `engine_name` | Nullable(String) | recommended |
| `policy.engine.version` | `engine_version` | Nullable(String) | recommended |
| `policy.rule.id` | `rule_id` | String | required |
| `policy.rule.name` | `rule_name` | Nullable(String) | opt_in |
| `policy.rule.uri` | `rule_uri` | Nullable(String) | recommended |
| `policy.evaluation.result` | `eval_result` | Enum8 | required |
| `policy.evaluation.message` | `eval_message` | Nullable(String) | opt_in |
| `policy.target.id` | `target_id` | String | recommended |
| `policy.target.name` | `target_name` | Nullable(String) | recommended |
| `policy.target.type` | `target_type` | Nullable(String) | recommended |
| `policy.target.environment` | `target_env` | Nullable(String) | recommended |

### Compliance Assessment Attributes (`registry.compliance`)

| Semconv Attribute | ClickHouse Column | Type | Requirement |
|:------------------|:------------------|:-----|:------------|
| `compliance.control.id` | `control_id` | String DEFAULT '' | required |
| `compliance.control.category` | `control_category` | Nullable(String) | recommended |
| `compliance.control.catalog.id` | `control_catalog_id` | Nullable(String) | required |
| `compliance.control.applicability` | `control_applicability` | Array(String) | opt_in |
| `compliance.frameworks` | `frameworks` | Array(String) | recommended |
| `compliance.requirements` | `requirements` | Array(String) | recommended |
| `compliance.status` | `compliance_status` | Enum8 | required |
| `compliance.risk.level` | `risk_level` | Nullable(Enum8) | opt_in |
| `compliance.remediation.action` | `remediation_action` | Nullable(Enum8) | opt_in |
| `compliance.remediation.status` | `remediation_status` | Nullable(Enum8) | opt_in |
| `compliance.remediation.description` | `remediation_desc` | Nullable(String) | opt_in |
| `compliance.remediation.exception.id` | `exception_id` | Nullable(String) | opt_in |
| `compliance.remediation.exception.active` | `exception_active` | Nullable(Bool) | opt_in |
| `compliance.assessment.id` | `evidence_id` | String | recommended |
| `compliance.enrichment.status` | `enrichment_status` | Enum8 | required |

### Identity / Timestamps

| Source | ClickHouse Column | Type |
|:-------|:------------------|:-----|
| OTel LogRecord timestamp | `collected_at` | DateTime64(3) |
| Insert time | `ingested_at` | DateTime64(3) DEFAULT now64(3) |
| Materialized | `row_key` | concat(evidence_id, '/', control_id, '/', requirement_id) |

## Proposed Semconv Additions

The current `beacon.evidence` entity is missing attributes required for Gemara audit-grade evidence. These extend the `registry.compliance` group without creating a separate namespace.

| Proposed Attribute | Type | ClickHouse Column | Rationale |
|:-------------------|:-----|:------------------|:----------|
| `compliance.policy.id` | string | `policy_id` (String DEFAULT '') | Links evidence to the Gemara L3 Policy driving the assessment. Required for the assistant to scope audit queries (`WHERE policy_id = ?`). |
| `compliance.assessment.requirement.id` | string | `requirement_id` (String DEFAULT '') | Assessment granularity below the control level. Gemara EvaluationLogs produce one AssessmentLog per requirement — this is the atomic unit the AuditLog maps to. |
| `compliance.assessment.plan.id` | string | `plan_id` | Ties an assessment to a specific plan within the Policy's `adherence.assessment-plans[]`. Required for cadence validation (did assessments occur at the expected frequency?). |
| `compliance.assessment.confidence` | enum (Undetermined, Low, Medium, High) | `confidence` | Confidence level of the assessment result. Used by the gap-analyst to classify evidence strength and by cross-framework coverage analysis via MappingDocument strength scores. |
| `compliance.assessment.steps` | int | `steps_executed` | Number of evaluation steps executed during the assessment. Provides assessment depth context — a single-step check vs. a multi-step validation suite. |

### Populated By

| Attribute | Path A (complyctl/ProofWatch) | Path B (truthbeam enrichment) |
|:----------|:------------------------------|:------------------------------|
| `compliance.policy.id` | Emitted by ProofWatch (complyctl knows the policy) | Mapped by truthbeam from rule→control→policy chain |
| `compliance.assessment.requirement.id` | Emitted by ProofWatch (complyctl evaluates per requirement) | Mapped by truthbeam if control→requirement mapping exists |
| `compliance.assessment.plan.id` | Emitted by ProofWatch (complyctl drives assessment plans) | Mapped by truthbeam if plan context available |
| `compliance.assessment.confidence` | Emitted by ProofWatch | Set by truthbeam based on mapping confidence |
| `compliance.assessment.steps` | Emitted by ProofWatch | Typically NULL for raw policy engine signals |

### Enrichment Provenance

The `compliance.enrichment.status` attribute tracks how compliance context was populated:

| Value | Meaning |
|:------|:--------|
| `Success` | Full compliance context available (source-provided or successfully enriched) |
| `Partial` | Some compliance attributes mapped, others missing |
| `Unmapped` | No mapping found for the policy rule — compliance context absent |
| `Unknown` | Enrichment status not determined |
| `Skipped` | Enrichment was not attempted (e.g., passthrough mode) |
