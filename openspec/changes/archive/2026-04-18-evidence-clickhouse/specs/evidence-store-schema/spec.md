## ADDED Requirements

### Requirement: evaluation_logs table stores flattened L5 assessment results

The `evaluation_logs` table SHALL store one row per `AssessmentLog` entry from a Gemara `EvaluationLog` artifact. Each row SHALL carry denormalized context fields (`log_id`, `target_id`, `target_env`, `policy_id`, `catalog_ref_id`, `control_id`, `control_name`, `control_result`) alongside assessment-specific fields (`requirement_id`, `plan_id`, `assessment_result`, `message`, `description`, `applicability`, `steps_executed`, `confidence_level`, `recommendation`, `collected_at`, `completed_at`). An `ingested_at` column SHALL record insertion time.

#### Scenario: Single EvaluationLog with two controls, three assessments total

- **WHEN** an EvaluationLog contains ControlEvaluation "AC-1" with 2 AssessmentLogs and ControlEvaluation "AC-2" with 1 AssessmentLog
- **THEN** 3 rows are inserted into `evaluation_logs`, each carrying the parent control's `control_id`, `control_name`, and `control_result`

#### Scenario: Query by policy and target

- **WHEN** agent executes `SELECT * FROM evaluation_logs WHERE policy_id = 'policy-xyz' AND target_id = 'cluster-prod' ORDER BY control_id, requirement_id`
- **THEN** results return all assessment rows for that policy-target pair, ordered for sequential processing by the Gap Analyst

### Requirement: enforcement_actions table stores flattened L6 action results

The `enforcement_actions` table SHALL store one row per `AssessmentFinding` within an `ActionResult` from a Gemara `EnforcementLog` artifact. Each row SHALL carry denormalized context fields (`log_id`, `target_id`, `target_env`, `policy_id`, `catalog_ref_id`, `control_id`, `requirement_id`) alongside action-specific fields (`disposition`, `method_id`, `assessment_result`, `eval_log_ref`, `message`, `has_exception`, `exception_refs`, `started_at`, `completed_at`). An `ingested_at` column SHALL record insertion time.

#### Scenario: EnforcementLog with one action containing two assessment findings

- **WHEN** an EnforcementLog contains one ActionResult with disposition "Enforced" and justification containing 2 AssessmentFindings for requirements "AC-1.1" and "AC-1.2"
- **THEN** 2 rows are inserted into `enforcement_actions`, each carrying the parent action's `disposition`, `method_id`, `started_at`, and `completed_at`

#### Scenario: Query enforcement actions joined to evaluation context

- **WHEN** agent executes `SELECT * FROM enforcement_actions WHERE policy_id = 'policy-xyz' AND target_id = 'cluster-prod' ORDER BY control_id, requirement_id`
- **THEN** results return all enforcement action rows for that policy-target pair, matchable to `evaluation_logs` rows by `control_id` + `requirement_id`

### Requirement: Tables use MergeTree engine with monthly partitioning

Both tables SHALL use the `MergeTree` engine. `evaluation_logs` SHALL be partitioned by `toYYYYMM(collected_at)`. `enforcement_actions` SHALL be partitioned by `toYYYYMM(started_at)`. Both tables SHALL define a sort key of `(target_id, policy_id, control_id, <time_column>)`.

#### Scenario: Partition pruning on time-range query

- **WHEN** agent queries `WHERE collected_at >= '2026-03-01' AND collected_at < '2026-04-01'`
- **THEN** ClickHouse reads only the `202603` partition

### Requirement: Tables define a TTL policy for automatic data expiry

Both tables SHALL define a TTL expression that drops rows older than a configurable retention period. The default retention period SHALL be 24 months.

#### Scenario: Rows older than retention period are dropped

- **WHEN** a row's `collected_at` (or `started_at`) is older than 24 months
- **THEN** ClickHouse automatically removes the row during the next merge cycle
