## MODIFIED Requirements

### Requirement: Ingestion loader writes to the `evidence` table
The `cmd/ingest` loader SHALL write flattened Gemara EvaluationLog and EnforcementLog data to the single `evidence` table instead of separate `evaluation_logs` and `enforcement_actions` tables.

#### Scenario: Ingest EvaluationLog YAML
- **WHEN** `cmd/ingest` receives a valid Gemara EvaluationLog YAML file
- **THEN** each AssessmentLog entry is flattened into one row in the `evidence` table
- **THEN** remediation columns are NULL

#### Scenario: Ingest EnforcementLog YAML
- **WHEN** `cmd/ingest` receives a valid Gemara EnforcementLog YAML file
- **THEN** each enforcement action is flattened into one row in the `evidence` table
- **THEN** both evaluation and remediation columns are populated

#### Scenario: Ingest combined evidence
- **WHEN** `cmd/ingest` receives an EnforcementLog that references evaluation results
- **THEN** the resulting row contains co-located evaluation and remediation data in a single `evidence` row

### Requirement: Ingestion loader sets enrichment provenance
The `cmd/ingest` loader SHALL set `enrichment_status` to indicate the data source.

#### Scenario: Gemara YAML ingested directly
- **WHEN** `cmd/ingest` loads a Gemara artifact
- **THEN** `enrichment_status` is set to `Success` (source provided full compliance context)

### Requirement: Ingestion loader is a local development tool
The `cmd/ingest` loader SHALL remain functional without requiring the OTel Collector stack.

#### Scenario: Local development without OTel
- **WHEN** a developer has ClickHouse running locally (or port-forwarded) but no OTel Collector
- **THEN** `cmd/ingest` writes directly to ClickHouse via native protocol
- **THEN** the `evidence` table is populated identically to the OTel pipeline path

#### Scenario: New columns populated by ingest
- **WHEN** `cmd/ingest` writes to the `evidence` table
- **THEN** new columns from the merged schema (`engine_name`, `target_type`, `risk_level`, `frameworks`, `requirements`) are populated where the Gemara YAML provides equivalent data, and NULL otherwise
