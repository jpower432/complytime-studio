## ADDED Requirements

### Requirement: Plugin exports metadata function
Every ingestor plugin SHALL export a `metadata()` function returning an `IngestorMetadata` struct with fields: `name` (string), `version` (string), `description` (string), `input_formats` ([]string, MIME types), `semconv_version` (string), `author` (string).

#### Scenario: Host reads plugin metadata
- **WHEN** the runtime loads a `.wasm` module
- **THEN** the runtime SHALL call `metadata()` and register the plugin's name, version, and supported input formats

#### Scenario: Plugin metadata missing required fields
- **WHEN** a plugin's `metadata()` returns a struct with `name` or `version` empty
- **THEN** the runtime SHALL reject the plugin with a validation error

### Requirement: Plugin exports transform function
Every ingestor plugin SHALL export a `transform(input: bytes) → Result<EvidenceBatch, Error>` function. The function receives raw evidence bytes and returns a batch of semconv-aligned evidence rows or an error.

#### Scenario: Successful transform
- **WHEN** a plugin receives valid Nessus XML bytes
- **THEN** the plugin SHALL return an `EvidenceBatch` with one `EvidenceRow` per finding, each containing at minimum: `evidence_id`, `target_id`, `rule_id`, `eval_result`, `collected_at`

#### Scenario: Unparseable input
- **WHEN** a plugin receives bytes it cannot parse (wrong format, corrupted)
- **THEN** the plugin SHALL return an error with a descriptive message, not panic or hang

### Requirement: EvidenceRow output schema is semconv-aligned
The `EvidenceRow` struct output by plugins SHALL align with the `beacon.evidence` OTel semantic convention. Required fields: `evidence_id`, `target_id`, `rule_id`, `eval_result`, `collected_at`. Recommended fields: `target_name`, `target_type`, `target_env`, `engine_name`, `engine_version`, `rule_name`, `eval_message`. Optional fields: `labels` (map<string,string>), `raw_ref` (string pointer to original document).

#### Scenario: Plugin omits required field
- **WHEN** a plugin returns an `EvidenceRow` without `evidence_id`
- **THEN** the host SHALL reject that row and increment the `total_skipped` counter

#### Scenario: Plugin includes labels
- **WHEN** a plugin returns an `EvidenceRow` with `labels: {"scan-type": "authenticated", "quarter": "Q2-2026"}`
- **THEN** the host SHALL preserve labels through enrichment and index them as searchable fields in OpenSearch

### Requirement: Host provides log import to plugins
The host SHALL expose a `log(level: u32, message_ptr: u32, message_len: u32)` function to plugins via WASI imports. Plugins SHALL use this for structured logging (parse warnings, skip reasons). Log output SHALL be captured by the host and associated with the ingest request.

#### Scenario: Plugin logs a parse warning
- **WHEN** a plugin calls `log(WARN, "skipping malformed finding at line 42")`
- **THEN** the host SHALL capture the message and include it in the `EvidenceBatch.warnings` array

### Requirement: Plugins do NOT perform compliance enrichment
Plugins SHALL NOT set compliance context fields: `policy_id`, `control_id`, `requirement_id`, `plan_id`, `compliance_status`, `enrichment_status`. These fields SHALL be set exclusively by the host-side enrichment pipeline. If a plugin sets these fields, the host SHALL overwrite them.

#### Scenario: Plugin sets control_id
- **WHEN** a plugin returns an `EvidenceRow` with `control_id = "AC-2"`
- **THEN** the host SHALL overwrite `control_id` with the value from its own enrichment lookup (or NULL if unmapped)
