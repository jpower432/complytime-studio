## ADDED Requirements

### Requirement: Server-side AuditLog enrichment
The `createAuditLogHandler` SHALL parse the `content` field using `gemara.ParseAuditLog` and populate `audit_start`, `audit_end`, `summary`, `framework`, and target metadata from the parsed result. The handler SHALL accept a simplified request body containing only `policy_id` and `content`.

#### Scenario: Valid AuditLog content submitted
- **WHEN** a POST to `/api/audit-logs` contains `{"policy_id":"my-policy","content":"<valid #AuditLog YAML>"}`
- **THEN** the handler SHALL parse the YAML, extract dates and classification counts, store the audit log with a JSON summary `{"strengths":N,"findings":N,"gaps":N,"observations":N}`, and return `201 Created`

#### Scenario: Invalid AuditLog YAML submitted
- **WHEN** a POST to `/api/audit-logs` contains `content` that fails `ParseAuditLog`
- **THEN** the handler SHALL return `400 Bad Request` with an error message describing the parse failure

#### Scenario: Missing content field
- **WHEN** a POST to `/api/audit-logs` contains an empty or missing `content` field
- **THEN** the handler SHALL return `400 Bad Request` with message "policy_id and content required"

### Requirement: Backward-compatible request body
The handler SHALL accept the new simplified body (`policy_id` + `content` only). Previously required fields `audit_start`, `audit_end`, and `summary` SHALL be ignored if present — the server always derives them from content.

#### Scenario: Old-format request with explicit dates
- **WHEN** a POST includes `audit_start`, `audit_end`, and `summary` alongside `content`
- **THEN** the handler SHALL ignore the client-provided values and use the parsed values from `content`
