## ADDED Requirements

### Requirement: Mapping entries table exists in ClickHouse
The system SHALL create a `mapping_entries` table during schema initialization with columns: `mapping_id`, `policy_id`, `control_id`, `requirement_id`, `framework`, `reference`, `strength`, `confidence`, `imported_at`.

#### Scenario: Fresh deployment
- **WHEN** the gateway starts and runs schema initialization
- **THEN** the `mapping_entries` table SHALL exist in ClickHouse with `ReplacingMergeTree(imported_at)` engine and `ORDER BY (policy_id, framework, control_id, reference)`

### Requirement: Mapping import parses YAML into structured entries
The system SHALL parse the `content` field of an imported mapping document and write one row to `mapping_entries` for each `(source, target)` pair in the `mappings` array.

#### Scenario: Import mapping with multiple targets per control
- **WHEN** a mapping document is imported via `POST /api/mappings/import` with a control that maps to 2 framework objectives
- **THEN** 2 rows SHALL be inserted into `mapping_entries`, one per target reference, each with the correct `control_id`, `framework`, `reference`, `strength`, and `confidence`

#### Scenario: Import mapping with missing optional fields
- **WHEN** a mapping entry has no `strength` or `confidence-level` field
- **THEN** the row SHALL be inserted with `strength` defaulting to 0 and `confidence` defaulting to empty string

#### Scenario: YAML parsing failure
- **WHEN** the `content` field contains invalid YAML or does not match the expected Gemara mapping structure
- **THEN** the mapping document blob SHALL still be stored in `mapping_documents`
- **THEN** a warning SHALL be logged
- **THEN** no rows SHALL be inserted into `mapping_entries`
- **THEN** the API SHALL return HTTP 201 (the blob import succeeded)

### Requirement: Re-import replaces existing entries
The system SHALL use `ReplacingMergeTree` deduplication so that re-importing the same mapping document replaces previous entries after merge.

#### Scenario: Re-import updated mapping
- **WHEN** a mapping document with the same `policy_id`, `framework`, `control_id`, and `reference` is imported again with a different `strength`
- **THEN** after ClickHouse merge, only the latest row (by `imported_at`) SHALL remain

### Requirement: Retroactive population on startup
The system SHALL populate `mapping_entries` from existing `mapping_documents` during schema initialization if entries are missing.

#### Scenario: Upgrade from pre-mapping-entries schema
- **WHEN** the gateway starts and `mapping_documents` contains rows but `mapping_entries` is empty
- **THEN** the system SHALL parse all existing mapping documents and insert structured entries

#### Scenario: Entries already populated
- **WHEN** the gateway starts and `mapping_entries` already has rows for all mapping documents
- **THEN** the system SHALL skip retroactive population
