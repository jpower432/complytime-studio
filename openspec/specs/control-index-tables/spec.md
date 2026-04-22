## ADDED Requirements

### Requirement: Controls table stores parsed ControlCatalog entries
The system SHALL define a `controls` ClickHouse table with columns: `catalog_id`, `control_id`, `title`, `objective`, `group_id`, `state`, `policy_id`, `imported_at`. The table SHALL use `ReplacingMergeTree(imported_at)` with `ORDER BY (catalog_id, control_id)`.

#### Scenario: ControlCatalog with three controls
- **WHEN** a ControlCatalog YAML with `metadata.id = "cc-soc2"` and 3 controls is imported
- **THEN** the `controls` table SHALL contain 3 rows with `catalog_id = "cc-soc2"`, each with the correct `control_id`, `title`, `objective`, `group_id`, and `state`

#### Scenario: Duplicate import is deduplicated
- **WHEN** the same ControlCatalog is imported twice
- **THEN** `ReplacingMergeTree` SHALL retain only the most recent version of each `(catalog_id, control_id)` pair

### Requirement: Assessment requirements table stores parsed assessment requirements
The system SHALL define an `assessment_requirements` ClickHouse table with columns: `catalog_id`, `control_id`, `requirement_id`, `text`, `applicability`, `recommendation`, `state`, `imported_at`. The `applicability` column SHALL use `Array(String)`. The table SHALL use `ReplacingMergeTree(imported_at)` with `ORDER BY (catalog_id, control_id, requirement_id)`.

#### Scenario: Control with two assessment requirements
- **WHEN** a ControlCatalog control has 2 assessment requirements with `applicability: ["all-environments"]`
- **THEN** the `assessment_requirements` table SHALL contain 2 rows with the correct `control_id`, `requirement_id`, `text`, and `applicability` array

#### Scenario: JOIN from evidence to requirement text
- **WHEN** `SELECT ar.text FROM evidence e JOIN assessment_requirements ar ON ar.control_id = e.control_id AND ar.requirement_id = e.requirement_id` is executed
- **THEN** the result SHALL include the assessment requirement text for each evidence row

### Requirement: Control-threats junction table stores control-to-threat cross-references
The system SHALL define a `control_threats` ClickHouse table with columns: `catalog_id`, `control_id`, `threat_reference_id`, `threat_entry_id`, `imported_at`. The table SHALL use `ReplacingMergeTree(imported_at)` with `ORDER BY (catalog_id, control_id, threat_reference_id, threat_entry_id)`.

#### Scenario: Control referencing two threats
- **WHEN** a control has `threats: [{reference-id: "tc-ref", entries: [{reference-id: "tc-ref", entry-id: "T-1"}, {reference-id: "tc-ref", entry-id: "T-2"}]}]`
- **THEN** the `control_threats` table SHALL contain 2 rows linking the control to `T-1` and `T-2`

#### Scenario: Threat-to-evidence traversal via JOIN
- **WHEN** `SELECT e.* FROM control_threats ct JOIN evidence e ON e.control_id = ct.control_id WHERE ct.threat_entry_id = 'T-3'` is executed
- **THEN** the result SHALL include all evidence rows for controls that address threat `T-3`

### Requirement: DDL added to EnsureSchema
The `controls`, `assessment_requirements`, and `control_threats` CREATE TABLE statements SHALL be added to `internal/clickhouse/client.go` `EnsureSchema` method. All statements SHALL use `IF NOT EXISTS`.

#### Scenario: Fresh database startup
- **WHEN** the gateway starts against an empty ClickHouse database
- **THEN** `EnsureSchema` SHALL create the `controls`, `assessment_requirements`, and `control_threats` tables without error

### Requirement: Backfill controls on startup
The system SHALL provide a `PopulateControls` function in `internal/store/populate.go` that iterates all stored catalog content, skips catalogs that already have `controls` rows, parses remaining catalogs, and inserts structured rows.

#### Scenario: Existing catalog with no control rows
- **WHEN** a ControlCatalog exists in the `policies` table but has no rows in `controls`
- **THEN** `PopulateControls` SHALL parse the catalog content and insert control, assessment requirement, and control-threat rows

#### Scenario: Catalog already populated
- **WHEN** a ControlCatalog already has rows in the `controls` table
- **THEN** `PopulateControls` SHALL skip parsing for that catalog
