## ADDED Requirements

### Requirement: PostgreSQL schema models the Gemara knowledge graph
The system SHALL maintain a PostgreSQL schema with tables for all Gemara artifact types: `policies`, `control_catalogs`, `controls`, `assessment_requirements`, `threat_catalogs`, `threats`, `risk_catalogs`, `risks`, `guidance_catalogs`, `guidelines`, `mapping_documents`, `mapping_entries`, `catalogs`. Foreign key constraints SHALL enforce referential integrity between related tables.

#### Scenario: Control references a valid catalog
- **WHEN** a control row is inserted with `catalog_id = "cat-01"`
- **THEN** a corresponding row in `control_catalogs` with `id = "cat-01"` MUST exist or the insert SHALL fail with a FK violation

#### Scenario: Risk-to-threat traversal via CTE
- **WHEN** the agent queries risks exposed by failing evidence for a policy
- **THEN** the system SHALL support a single recursive CTE traversing `evidence → control_threats → risk_threats → risks` without multi-query composition

### Requirement: JSONB storage for raw artifact content
Each artifact table SHALL include a `content` column of type `JSONB` storing the full parsed Gemara artifact. The system SHALL support JSONB path queries (e.g., `content->'adherence'->'assessment-plans'`) for ad-hoc extraction without application-side YAML parsing.

#### Scenario: Query assessment plans from policy content
- **WHEN** the agent queries `SELECT content->'adherence'->'assessment-plans' FROM policies WHERE id = $1`
- **THEN** PostgreSQL SHALL return the assessment plans array as a JSON value

### Requirement: Materialized assessment_plan_status table
The system SHALL maintain an `assessment_plan_status` table with columns: `policy_id`, `plan_id`, `target_id`, `last_evidence_at`, `engine_name`, `source_match` (boolean), `cadence_status` (current/stale/blind), `latest_result`, `classification` (Healthy/Failing/Wrong Source/Stale/Blind). This table SHALL be updated on every evidence ingest.

#### Scenario: Evidence ingest updates posture
- **WHEN** new evidence rows are ingested for plan AP-03 on target prod-cluster
- **THEN** the `assessment_plan_status` row for (policy, AP-03, prod-cluster) SHALL be updated with the latest evidence timestamp, source match result, and classification

#### Scenario: Agent reads posture directly
- **WHEN** the agent performs a posture check for a policy
- **THEN** the agent SHALL query `assessment_plan_status` with a single `SELECT` instead of computing posture from raw evidence

### Requirement: Rule-to-control mapping table for enrichment
The system SHALL maintain a `rule_mappings` table mapping external scanner `rule_id` values to Gemara `control_id` and `requirement_id`. This table SHALL be the primary lookup for compliance enrichment during evidence ingestion.

#### Scenario: Enrichment lookup succeeds
- **WHEN** the pipeline receives evidence with `rule_id = "nessus-12345"`
- **THEN** the system SHALL look up `rule_mappings` and return `control_id`, `requirement_id`, and `plan_id` for enrichment

#### Scenario: Enrichment lookup fails
- **WHEN** no mapping exists for a `rule_id`
- **THEN** the system SHALL set `enrichment_status = 'Unmapped'` on the evidence row and proceed without blocking

### Requirement: Schema managed via versioned migrations
PostgreSQL schema changes SHALL be managed via numbered migration files applied at Gateway startup. The system SHALL track applied migrations in a `schema_migrations` table. Migrations SHALL be idempotent.

#### Scenario: Gateway starts with pending migrations
- **WHEN** the Gateway starts and detects unapplied migrations
- **THEN** the system SHALL apply them in order and record each in `schema_migrations`

#### Scenario: Gateway starts with current schema
- **WHEN** all migrations are already applied
- **THEN** the system SHALL skip migration and proceed to serve requests
