## ADDED Requirements

### Requirement: Single `evidence` table replaces `evaluation_logs` and `enforcement_actions`
The ClickHouse schema SHALL define a single `evidence` table that co-locates evaluation and remediation data.

#### Scenario: Evaluation-only record
- **WHEN** evidence is ingested with `eval_result` populated and no remediation attributes
- **THEN** the record is inserted into the `evidence` table with remediation columns as NULL

#### Scenario: Evaluation with remediation record
- **WHEN** evidence is ingested with both `eval_result` and `remediation_action` populated
- **THEN** a single row in the `evidence` table contains both evaluation and remediation data

#### Scenario: Query returns co-located data
- **WHEN** the gap-analyst queries `SELECT * FROM evidence WHERE target_id = ? AND policy_id = ?`
- **THEN** results include both evaluation results and remediation actions in the same row without requiring a JOIN

### Requirement: Table uses ReplacingMergeTree with deduplication
The `evidence` table SHALL use `ReplacingMergeTree` to handle duplicate ingestion.

#### Scenario: Duplicate record ingested
- **WHEN** the same evidence record is ingested twice (same `evidence_id`, `control_id`, `requirement_id`)
- **THEN** the `ReplacingMergeTree` engine retains only the most recent version based on `ingested_at`

### Requirement: Table is partitioned by month with TTL auto-expiry
The `evidence` table SHALL partition by `toYYYYMM(collected_at)` and apply a configurable TTL.

#### Scenario: Retention period expires
- **WHEN** a partition's `collected_at` exceeds the configured retention period (default: 24 months)
- **THEN** ClickHouse automatically drops the expired partition

#### Scenario: Retention period is configurable via Helm values
- **WHEN** `clickhouse.retentionMonths` is set in Helm values
- **THEN** the DDL TTL clause uses the configured value

### Requirement: Sort key optimizes the audit query pattern
The `evidence` table sort key SHALL be `(target_id, policy_id, control_id, collected_at, row_key)`.

#### Scenario: Gap-analyst audit query
- **WHEN** the gap-analyst queries evidence filtered by `target_id`, `policy_id`, and a `collected_at` time range
- **THEN** ClickHouse uses the sort key for efficient range scans without full table scans

### Requirement: Old tables are removed from the schema ConfigMap
The Helm chart schema ConfigMap SHALL contain only the `evidence` table DDL.

#### Scenario: Helm template renders new schema
- **WHEN** `helm template` is run with `clickhouse.enabled=true`
- **THEN** the `init.sql` ConfigMap contains `CREATE TABLE IF NOT EXISTS evidence (...)` 
- **THEN** no `evaluation_logs` or `enforcement_actions` DDL is present
