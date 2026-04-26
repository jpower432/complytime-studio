# Source registry tracking — delta specs

## ADDED Requirements

### Requirement: evidence table includes source_registry
The `evidence` table MUST include a column `source_registry` of type `Nullable(String)`.

#### Scenario: Column exists after migration
- **GIVEN** the gateway has run `EnsureSchema` through the migration that adds `source_registry`
- **WHEN** the `evidence` table DDL is inspected
- **THEN** a nullable string column `source_registry` is present

#### Scenario: New rows may set or omit the column
- **GIVEN** a consumer inserts an evidence row
- **WHEN** `source_registry` is omitted or set to `NULL`
- **THEN** the row stores successfully and `source_registry` is `NULL`

#### Scenario: OCI registry hostname is storable
- **GIVEN** a producer supplies a non-empty OCI registry hostname or URL
- **WHEN** the value is written to `source_registry`
- **THEN** the stored value matches the supplied string (no loss of precision for string content)

### Requirement: REST evidence handler accepts source_registry
The REST evidence handler MUST accept an optional `source_registry` field in the request body and persist it to ClickHouse on insert.

#### Scenario: JSON payload includes source_registry
- **GIVEN** a `POST` to the evidence API with a valid body containing `source_registry`
- **WHEN** the request completes successfully
- **THEN** the inserted row in `evidence` has `source_registry` equal to the request value (or `NULL` if explicitly null)

#### Scenario: Omitted field remains NULL
- **GIVEN** a `POST` to the evidence API without a `source_registry` key
- **WHEN** the request completes successfully
- **THEN** the inserted row has `source_registry` `NULL`

#### Scenario: Reject invalid type per API contract
- **GIVEN** a `POST` with `source_registry` of a type the handler does not support (e.g. object instead of string)
- **WHEN** validation runs
- **THEN** the handler MUST return a client error and MUST NOT partially persist an invalid `source_registry` value

### Requirement: OTel compliance.source.registry maps to source_registry
The OTel attribute `compliance.source.registry` MUST map to the ClickHouse `evidence.source_registry` column for ingests that use the collector→ClickHouse path.

#### Scenario: Attributed log maps to column
- **GIVEN** an evidence log record with `compliance.source.registry` set
- **WHEN** the log is written to the `evidence` table
- **THEN** `source_registry` contains the same string as the attribute value

#### Scenario: Unset attribute yields NULL
- **GIVEN** an evidence log record without `compliance.source.registry`
- **WHEN** the log is written to the `evidence` table
- **THEN** `source_registry` is `NULL`

#### Scenario: Documented in semconv alignment
- **GIVEN** the evidence semconv alignment document is the canonical mapping reference
- **WHEN** a reader looks up `compliance.source.registry`
- **THEN** the document states the `evidence` column name `source_registry` and type `Nullable(String)`

### Requirement: Workbench shows source_registry on evidence detail
The Workbench evidence detail view MUST display `source_registry` when the row has a non-NULL, non-empty value.

#### Scenario: Value visible to user
- **GIVEN** an evidence record with `source_registry` set
- **WHEN** the user opens the evidence detail view for that row
- **THEN** the UI shows the registry value (e.g. label plus copy or link affordance as implemented)

#### Scenario: NULL column hidden or clearly empty
- **GIVEN** an evidence record with `source_registry` `NULL` or empty
- **WHEN** the user opens the evidence detail view
- **THEN** the UI does not assert a false registry, and MAY omit the field or show an empty/unknown state per UX convention

#### Scenario: List view remains usable
- **GIVEN** the evidence list view exists
- **WHEN** the user navigates from list to detail
- **THEN** `source_registry` is available in detail without requiring a separate query shape that omits the column

### Requirement: attestation-verification uses source_registry for cross-registry pulls
The attestation-verification skill SHOULD use `source_registry` (when present) to resolve the OCI registry for pulling `attestation_ref` bundles, instead of assuming a single default registry for all evidence rows.

#### Scenario: Non-default registry when source_registry is set
- **GIVEN** an evidence row with both `attestation_ref` and `source_registry` populated
- **WHEN** the skill requests a bundle via oras-mcp
- **THEN** the pull targets the registry indicated by `source_registry` (or passes equivalent registry context to the tool, per implementation)

#### Scenario: Default behavior when NULL
- **GIVEN** an evidence row with `attestation_ref` but `source_registry` is `NULL`
- **WHEN** the skill pulls the attestation bundle
- **THEN** the skill uses the same default registry resolution as today (e.g. Studio-configured or policy context)

#### Scenario: Mismatch is explainable
- **GIVEN** a pull fails because the registry in `source_registry` is unreachable or unauthorized
- **WHEN** the skill reports the outcome
- **THEN** the verdict or message references registry resolution and does not claim verification success

### Requirement: Studio boundary contract does not require raw evidence payloads
Studio MUST NOT require raw evidence payloads in ClickHouse. Summary metadata, identifiers, and OCI digests (including `attestation_ref` and `source_registry`) are sufficient to operate dashboards, traceability, and on-demand attestation verification.

#### Scenario: No schema gate on raw binary columns
- **GIVEN** the documented boundary contract
- **WHEN** a deployment ingest path sends only semconv summary fields and OCI references
- **THEN** Studio accepts and stores the row without requiring a blob, screenshot, or log dump column to be non-NULL

#### Scenario: Provenance does not require raw data in CH
- **GIVEN** an auditor needs to trace raw evidence
- **WHEN** they have `attestation_ref` and optional `source_registry`
- **THEN** the documented path is: retrieve the bundle from the boundary OCI registry using those references, not from ClickHouse raw payload storage

#### Scenario: Complyctl remains responsible for what crosses the boundary
- **GIVEN** the complyctl tool pushes attestation bundles to a boundary-scoped OCI registry
- **WHEN** it emits OTel or REST to Studio
- **THEN** the contract states summary rows and references cross the boundary; raw bundles remain at the registry under boundary access control

### Requirement: optional gateway warning for large eval_message
The Gateway MAY emit a warning (e.g. structured log) when an evidence row or ingest request contains an unusually large `eval_message` value that may indicate raw or embedded data rather than a short summary, without blocking ingestion.

#### Scenario: large message triggers optional warning
- **GIVEN** a configurable or fixed threshold for `eval_message` length
- **WHEN** a received row exceeds that threshold
- **THEN** the Gateway MAY log a warning identifying the issue class (suspected non-summary content)

#### Scenario: summary-sized message does not warn
- **GIVEN** `eval_message` within normal summary length
- **WHEN** the row is ingested
- **THEN** the optional warning path is not required to fire

#### Scenario: Ingestion succeeds regardless
- **GIVEN** a row with a very large `eval_message`
- **WHEN** the insert otherwise succeeds
- **THEN** the row is stored; warning behavior (if any) is observability-only and does not change HTTP success semantics unless a separate product decision adds rejection
