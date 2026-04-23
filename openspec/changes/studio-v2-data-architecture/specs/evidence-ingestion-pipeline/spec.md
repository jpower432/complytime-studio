## ADDED Requirements

### Requirement: Gateway accepts raw evidence with ingestor hint
The Gateway SHALL expose `POST /api/evidence/ingest` accepting raw bytes with headers: `X-Ingestor` (plugin name, required), `X-Policy-Id` (policy context, optional), `Content-Type` (MIME type for format validation). The endpoint SHALL resolve the ingestor plugin, execute the WASM transform, and feed output to the enrichment pipeline.

#### Scenario: Valid ingest request
- **WHEN** a request arrives with `X-Ingestor: nessus-xml`, `Content-Type: application/xml`, and valid XML body
- **THEN** the Gateway SHALL execute the "nessus-xml" plugin and proceed to enrichment

#### Scenario: Unknown ingestor
- **WHEN** a request arrives with `X-Ingestor: unknown-plugin`
- **THEN** the Gateway SHALL return 400 with message "ingestor 'unknown-plugin' not found"

#### Scenario: Content-Type mismatch
- **WHEN** a request arrives with `X-Ingestor: nessus-xml` but `Content-Type: text/csv`
- **THEN** the Gateway SHALL return 400 with message "ingestor 'nessus-xml' does not accept text/csv"

### Requirement: Compliance enrichment via PostgreSQL lookup
After WASM transform, the pipeline SHALL enrich each `EvidenceRow` by looking up `rule_id` in the `rule_mappings` table to resolve `control_id`, `requirement_id`, and `plan_id`. When `X-Policy-Id` is provided, the lookup SHALL be scoped to that policy. Enrichment status SHALL be set to `Success`, `Partial`, or `Unmapped` based on lookup results.

#### Scenario: Full enrichment
- **WHEN** `rule_mappings` contains a complete mapping for `rule_id = "nessus-12345"` under the given policy
- **THEN** the pipeline SHALL set `policy_id`, `control_id`, `requirement_id`, `plan_id`, and `enrichment_status = Success`

#### Scenario: No mapping exists
- **WHEN** `rule_mappings` has no entry for the given `rule_id`
- **THEN** the pipeline SHALL set `enrichment_status = Unmapped` and leave compliance fields NULL

### Requirement: Provenance gate validates evidence against assessment plans
After enrichment, the pipeline SHALL validate each row against the policy's assessment plan. For each enriched row with a resolved `plan_id`, the gate SHALL check: (1) `engine_name` matches the plan's `executor.id`, (2) `collected_at` is within the plan's frequency window. Each row SHALL be tagged with `gate_status` (accepted/flagged/rejected) and `gate_reason`.

#### Scenario: Evidence passes gate
- **WHEN** evidence has `engine_name = "nessus"` matching executor and `collected_at` within the quarterly window
- **THEN** the gate SHALL tag `gate_status = accepted`

#### Scenario: Wrong executor
- **WHEN** evidence has `engine_name = "qualys"` but the plan expects executor "nessus"
- **THEN** the gate SHALL tag `gate_status = flagged` and `gate_reason = "wrong_source: expected nessus, got qualys"`

#### Scenario: Unenriched evidence skips gate
- **WHEN** evidence has `enrichment_status = Unmapped` (no plan_id resolved)
- **THEN** the gate SHALL tag `gate_status = accepted` and skip provenance validation

### Requirement: Fan-out write to all three stores
After enrichment and gating, the pipeline SHALL write to all three stores: (1) PostgreSQL — upsert `evidence_summary` and update `assessment_plan_status`, (2) OpenSearch — index the full evidence document, (3) Bulk Store — append the raw evidence row. Write failures to OpenSearch or Bulk Store SHALL NOT block the request. Write failure to PostgreSQL SHALL fail the request.

#### Scenario: All writes succeed
- **WHEN** PG, OpenSearch, and Bulk Store writes all succeed
- **THEN** the endpoint SHALL return 200 with ingest stats (rows accepted, flagged, rejected)

#### Scenario: OpenSearch write fails
- **WHEN** PG write succeeds but OpenSearch write fails
- **THEN** the endpoint SHALL return 200 (PG is source of truth), log the OpenSearch error, and enqueue a retry

#### Scenario: PostgreSQL write fails
- **WHEN** PG write fails
- **THEN** the endpoint SHALL return 500 and not write to OpenSearch or Bulk Store
