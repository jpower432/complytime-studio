## REMOVED Requirements

### Requirement: CSV evidence import
**Reason**: Manual CSV import creates an unverifiable ingestion path that bypasses the certifier pipeline. All structured evidence flows through OTel Collector or `cmd/ingest` (Gemara artifacts).
**Migration**: Use `cmd/ingest` with Gemara `EvaluationLog` or `EnforcementLog` artifacts for bulk evidence ingestion.

### Requirement: Manual evidence form entry
**Reason**: Manual form entry produces unstructured evidence with no provenance, no attestation, and no engine identity. Certifiers cannot meaningfully assess hand-entered data.
**Migration**: Use `cmd/ingest` with Gemara artifacts. For ad-hoc evidence, produce a minimal `EvaluationLog` YAML and ingest it.

### Requirement: Evidence upload button in UI
**Reason**: The "Upload Evidence" button and associated modal are removed entirely from `evidence-view.tsx` — not conditionally hidden based on embedded/admin state, but removed from the component.
**Migration**: Evidence management is through structured ingestion pipelines (OTel, `cmd/ingest`), not through the workbench UI.

## ADDED Requirements

### Requirement: Upload endpoint returns gone
The `POST /api/evidence/upload` endpoint SHALL return HTTP 410 Gone with a response body directing callers to use `cmd/ingest`.

#### Scenario: Client calls removed endpoint
- **WHEN** a client sends a POST to `/api/evidence/upload`
- **THEN** the server SHALL respond with 410 Gone and body `{"error": "manual upload removed, use cmd/ingest with Gemara artifacts"}`

### Requirement: No upload UI elements
The evidence view SHALL NOT render any upload button, file input, or CSV import form regardless of user role or embedded state.

#### Scenario: Admin views evidence page
- **WHEN** an admin user navigates to the evidence page
- **THEN** no upload button or form SHALL be present

#### Scenario: Embedded evidence view
- **WHEN** the evidence view is embedded in the policy detail view
- **THEN** no upload button or form SHALL be present
