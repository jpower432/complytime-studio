## ADDED Requirements

### Requirement: Risks table stores parsed RiskCatalog entries
The system SHALL define a `risks` ClickHouse table with columns: `catalog_id`, `risk_id`, `title`, `description`, `severity`, `group_id`, `impact`, `policy_id`, `imported_at`. The `severity` column SHALL use `LowCardinality(String)`. The table SHALL use `ReplacingMergeTree(imported_at)` with `ORDER BY (catalog_id, risk_id)`.

#### Scenario: RiskCatalog with three risks
- **WHEN** a RiskCatalog YAML with `metadata.id = "rc-infra"` and 3 risks is imported
- **THEN** the `risks` table SHALL contain 3 rows with `catalog_id = "rc-infra"`, each with the correct `risk_id`, `title`, `description`, `severity`, `group_id`, and `impact`

#### Scenario: Severity values stored as strings
- **WHEN** a risk has `severity: Critical`
- **THEN** the `risks` row SHALL store `severity = 'Critical'`

#### Scenario: Duplicate import is deduplicated
- **WHEN** the same RiskCatalog is imported twice
- **THEN** `ReplacingMergeTree` SHALL retain only the most recent version of each `(catalog_id, risk_id)` pair

### Requirement: Risk-threats junction table stores risk-to-threat cross-references
The system SHALL define a `risk_threats` ClickHouse table with columns: `catalog_id`, `risk_id`, `threat_reference_id`, `threat_entry_id`, `imported_at`. The table SHALL use `ReplacingMergeTree(imported_at)` with `ORDER BY (catalog_id, risk_id, threat_reference_id, threat_entry_id)`.

#### Scenario: Risk referencing two threats
- **WHEN** a risk has `threats: [{reference-id: "tc-ref", entries: [{reference-id: "tc-ref", entry-id: "T-1"}, {reference-id: "tc-ref", entry-id: "T-2"}]}]`
- **THEN** the `risk_threats` table SHALL contain 2 rows linking the risk to `T-1` and `T-2`

#### Scenario: Join key matches control_threats
- **WHEN** `control_threats` has a row with `threat_entry_id = "T-1"` and `risk_threats` has a row with `threat_entry_id = "T-1"`
- **THEN** a JOIN on `threat_entry_id` SHALL produce the control-to-risk link through their common threat

### Requirement: DDL added to EnsureSchema
The `risks` and `risk_threats` CREATE TABLE statements SHALL be added to `internal/clickhouse/client.go` `EnsureSchema` method. All statements SHALL use `IF NOT EXISTS`.

#### Scenario: Fresh database startup
- **WHEN** the gateway starts against an empty ClickHouse database
- **THEN** `EnsureSchema` SHALL create the `risks` and `risk_threats` tables without error

### Requirement: ParseRiskCatalog extracts structured rows
The system SHALL provide a `ParseRiskCatalog` function in `internal/gemara/` that accepts RiskCatalog YAML content, a catalog ID, and a policy ID, and returns `[]RiskRow` and `[]RiskThreatRow`.

#### Scenario: Parse a complete RiskCatalog
- **WHEN** `ParseRiskCatalog` is called with valid RiskCatalog YAML containing 2 risks, the first linking to 3 threats
- **THEN** it SHALL return 2 `RiskRow` entries and 3 `RiskThreatRow` entries
- **THEN** each `RiskRow` SHALL include `severity` as a string (e.g., "High")

#### Scenario: Risk with no threats
- **WHEN** a risk has an empty `threats` array
- **THEN** `ParseRiskCatalog` SHALL return the `RiskRow` with zero corresponding `RiskThreatRow` entries

### Requirement: Import handler supports RiskCatalog type
The `importCatalogHandler` SHALL detect `metadata.type = "RiskCatalog"` and call `parseCatalogStructuredRows` to insert rows into `risks` and `risk_threats`.

#### Scenario: POST a RiskCatalog via API
- **WHEN** `POST /api/catalogs/import` receives a body with `content` containing a valid RiskCatalog
- **THEN** the handler SHALL insert a `catalogs` row with `catalog_type = "RiskCatalog"`
- **THEN** the handler SHALL insert structured rows into `risks` and `risk_threats`

### Requirement: Backfill risks on startup
The system SHALL provide a `PopulateRisks` function in `internal/store/populate.go` that iterates all stored catalog content, skips catalogs that already have `risks` rows, parses remaining catalogs, and inserts structured rows.

#### Scenario: Existing catalog with no risk rows
- **WHEN** a RiskCatalog exists in the `catalogs` table but has no rows in `risks`
- **THEN** `PopulateRisks` SHALL parse the catalog content and insert risk and risk-threat rows

#### Scenario: Catalog already populated
- **WHEN** a RiskCatalog already has rows in the `risks` table
- **THEN** `PopulateRisks` SHALL skip parsing for that catalog

### Requirement: RiskStore interface
The system SHALL define a `RiskStore` interface in `internal/store/store.go` with methods: `InsertRisks(ctx, []RiskRow)`, `InsertRiskThreats(ctx, []RiskThreatRow)`, `CountRisks(ctx, catalogID)`. The `Store` type SHALL implement this interface.

#### Scenario: Compile-time satisfaction check
- **WHEN** the code compiles
- **THEN** `var _ RiskStore = (*Store)(nil)` SHALL pass without error
