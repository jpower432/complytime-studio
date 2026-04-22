## ADDED Requirements

### Requirement: Gateway auto-persists AuditLog artifacts from A2A stream
The gateway A2A proxy SHALL intercept `TaskArtifactUpdateEvent` SSE events where any part has `metadata.mimeType` equal to `application/yaml`. When detected, the gateway SHALL parse the YAML content using `ParseAuditLog`, and if valid, persist the audit log to ClickHouse.

#### Scenario: Valid AuditLog artifact in stream
- **GIVEN** the agent produces a `TaskArtifactUpdateEvent` with a part containing valid `#AuditLog` YAML and `metadata.mimeType: application/yaml`
- **WHEN** the SSE event passes through the gateway A2A proxy
- **THEN** the gateway SHALL call `InsertAuditLog` with the parsed content, extracted provenance (`model`, `prompt_version`), and derived `policy_id`
- **AND** the SSE event SHALL be forwarded to the client unchanged

#### Scenario: Invalid YAML in artifact
- **GIVEN** the agent produces a `TaskArtifactUpdateEvent` with `metadata.mimeType: application/yaml` but the content fails `ParseAuditLog`
- **WHEN** the SSE event passes through the gateway A2A proxy
- **THEN** the gateway SHALL log a warning with the parse error
- **AND** the gateway SHALL NOT persist the artifact
- **AND** the SSE event SHALL be forwarded to the client unchanged

#### Scenario: Non-YAML artifact
- **GIVEN** the agent produces a `TaskArtifactUpdateEvent` without `metadata.mimeType: application/yaml`
- **WHEN** the SSE event passes through the gateway A2A proxy
- **THEN** the gateway SHALL ignore the event for persistence purposes
- **AND** the SSE event SHALL be forwarded to the client unchanged

### Requirement: Provenance metadata extracted from artifact parts
The gateway SHALL extract `model` and `promptVersion` from artifact part `metadata` and pass them as `Model` and `PromptVersion` to `InsertAuditLog`.

#### Scenario: Artifact carries provenance metadata
- **GIVEN** an artifact part has `metadata.model: "gemini-2.5-pro"` and `metadata.promptVersion: "a1b2c3d4e5f6"`
- **WHEN** the gateway auto-persists the artifact
- **THEN** the stored `audit_logs` row SHALL have `model = 'gemini-2.5-pro'` and `prompt_version = 'a1b2c3d4e5f6'`

#### Scenario: Artifact lacks provenance metadata
- **GIVEN** an artifact part does not contain `metadata.model` or `metadata.promptVersion`
- **WHEN** the gateway auto-persists the artifact
- **THEN** the stored `audit_logs` row SHALL have NULL `model` and `prompt_version`

### Requirement: Content-addressed audit_id for auto-persisted artifacts
The gateway SHALL compute `audit_id` as `sha256(content)[:16]` for auto-persisted artifacts. This ensures idempotent persistence — re-processing the same artifact content produces the same row key.

#### Scenario: Same artifact streamed twice
- **GIVEN** the agent produces the same AuditLog YAML content in two separate events
- **WHEN** both events are auto-persisted
- **THEN** both inserts SHALL use the same `audit_id`
- **AND** `ReplacingMergeTree` SHALL deduplicate to a single row after merge

### Requirement: Policy ID derivation
The gateway SHALL derive `policy_id` from the parsed AuditLog YAML content. If derivation fails, the gateway SHALL fall back to `"unassigned"` and log a warning.

#### Scenario: AuditLog contains policy reference
- **GIVEN** the parsed AuditLog YAML contains a resolvable policy or framework identifier
- **WHEN** the gateway auto-persists the artifact
- **THEN** `policy_id` SHALL be set to the derived value

#### Scenario: AuditLog lacks policy reference
- **GIVEN** the parsed AuditLog YAML does not contain a resolvable policy identifier
- **WHEN** the gateway auto-persists the artifact
- **THEN** `policy_id` SHALL be set to `"unassigned"`
- **AND** the gateway SHALL log a warning

### Requirement: Feature toggle
The gateway SHALL support an `AUTO_PERSIST_ARTIFACTS` environment variable. When set to `"false"`, the A2A proxy SHALL NOT intercept or persist artifacts. Default: `"true"`.

#### Scenario: Feature disabled
- **GIVEN** `AUTO_PERSIST_ARTIFACTS` is set to `"false"`
- **WHEN** the agent produces a `TaskArtifactUpdateEvent`
- **THEN** the gateway SHALL forward the event without any persistence logic

#### Scenario: Feature enabled (default)
- **GIVEN** `AUTO_PERSIST_ARTIFACTS` is unset or set to `"true"`
- **WHEN** the agent produces a valid AuditLog artifact
- **THEN** the gateway SHALL auto-persist the artifact

### Requirement: Async persistence does not block stream
The gateway SHALL persist artifacts asynchronously. Store failures SHALL be logged but SHALL NOT interrupt or delay the SSE stream to the client.

#### Scenario: Store insert fails
- **GIVEN** `InsertAuditLog` returns an error
- **WHEN** the gateway attempts to auto-persist an artifact
- **THEN** the gateway SHALL log the error at `ERROR` level
- **AND** the SSE stream SHALL continue without interruption
- **AND** the client SHALL still receive the artifact for manual save
