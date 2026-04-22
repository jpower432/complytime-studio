## ADDED Requirements

### Requirement: Threats table stores parsed ThreatCatalog entries
The system SHALL define a `threats` ClickHouse table with columns: `catalog_id`, `threat_id`, `title`, `description`, `group_id`, `policy_id`, `imported_at`. The table SHALL use `ReplacingMergeTree(imported_at)` with `ORDER BY (catalog_id, threat_id)`.

#### Scenario: ThreatCatalog with five threats
- **WHEN** a ThreatCatalog YAML with `metadata.id = "tc-cnsc"` and 5 threats is imported
- **THEN** the `threats` table SHALL contain 5 rows with `catalog_id = "tc-cnsc"`, each with the correct `threat_id`, `title`, `description`, and `group_id`

#### Scenario: Duplicate import is deduplicated
- **WHEN** the same ThreatCatalog is imported twice
- **THEN** `ReplacingMergeTree` SHALL retain only the most recent version of each `(catalog_id, threat_id)` pair

### Requirement: DDL added to EnsureSchema
The `threats` CREATE TABLE statement SHALL be added to `internal/clickhouse/client.go` `EnsureSchema` method using `IF NOT EXISTS`.

#### Scenario: Fresh database startup
- **WHEN** the gateway starts against an empty ClickHouse database
- **THEN** `EnsureSchema` SHALL create the `threats` table without error

### Requirement: Backfill threats on startup
The system SHALL provide a `PopulateThreats` function in `internal/store/populate.go` that iterates all stored catalog content, skips catalogs that already have `threats` rows, parses remaining catalogs, and inserts structured rows.

#### Scenario: Existing catalog with no threat rows
- **WHEN** a ThreatCatalog exists in the store but has no rows in `threats`
- **THEN** `PopulateThreats` SHALL parse the catalog content and insert threat rows

#### Scenario: Catalog already populated
- **WHEN** a ThreatCatalog already has rows in the `threats` table
- **THEN** `PopulateThreats` SHALL skip parsing for that catalog
