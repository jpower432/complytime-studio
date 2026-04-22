## SUPERSEDED

This spec described `cmd/ingest` as the evidence loader. That binary has been removed.

Evidence ingestion now uses two paths:
- **OTel pipeline**: complyctl/ProofWatch → OTel Collector → ClickHouse exporter → `evidence` table. See [OTel-Native Ingestion](../../../docs/decisions/otel-native-ingestion.md).
- **REST API**: `POST /api/evidence` (JSON) and `POST /api/evidence/upload` (CSV) for seeding, manual import, and non-OTel producers.

## ADDED Requirements

### Requirement: Catalog import handler parses ControlCatalog at ingest
The system SHALL extend or add an import handler that accepts ControlCatalog YAML, stores the raw content, and calls `ParseControlCatalog` to extract and insert structured rows into `controls`, `assessment_requirements`, and `control_threats` tables. Parse failures SHALL be logged as warnings without failing the import.

#### Scenario: ControlCatalog imported successfully
- **WHEN** a ControlCatalog YAML is imported with `metadata.id = "cc-soc2"` and 5 controls
- **THEN** raw content is stored AND 5 rows appear in `controls` AND corresponding rows in `assessment_requirements` and `control_threats`

#### Scenario: ControlCatalog parse failure
- **WHEN** a ControlCatalog import contains malformed YAML
- **THEN** the raw content is stored, a warning is logged, and structured rows are skipped

### Requirement: Catalog import handler parses ThreatCatalog at ingest
The system SHALL extend or add an import handler that accepts ThreatCatalog YAML, stores the raw content, and calls `ParseThreatCatalog` to extract and insert structured rows into the `threats` table. Parse failures SHALL be logged as warnings without failing the import.

#### Scenario: ThreatCatalog imported successfully
- **WHEN** a ThreatCatalog YAML is imported with `metadata.id = "tc-cnsc"` and 3 threats
- **THEN** raw content is stored AND 3 rows appear in `threats`

#### Scenario: ThreatCatalog parse failure
- **WHEN** a ThreatCatalog import contains malformed YAML
- **THEN** the raw content is stored, a warning is logged, and structured rows are skipped

### Requirement: Store interfaces for new tables
The system SHALL define `ControlStore` and `ThreatStore` interfaces in `internal/store/store.go` with insert, count, and query methods for the new tables. These interfaces SHALL be implemented by the ClickHouse store.

#### Scenario: ControlStore interface
- **WHEN** the `ControlStore` interface is compiled
- **THEN** it SHALL include `InsertControls`, `InsertAssessmentRequirements`, `InsertControlThreats`, `CountControls`, and `QueryControls` methods

#### Scenario: ThreatStore interface
- **WHEN** the `ThreatStore` interface is compiled
- **THEN** it SHALL include `InsertThreats`, `CountThreats`, and `QueryThreats` methods
