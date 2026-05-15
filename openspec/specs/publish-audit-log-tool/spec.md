### Requirement: Agent publishes draft AuditLog via MCP tool
The agent SHALL have a `save_draft_audit_log` MCP tool that persists a validated AuditLog YAML as a **draft** via the gateway REST API.

#### Scenario: Agent calls save_draft_audit_log with valid AuditLog
- **WHEN** the agent calls `save_draft_audit_log(policy_id, content, summary, ...)`
- **THEN** the tool SHALL POST to `POST /api/draft-audit-logs` on the gateway (`:8080`)
- **THEN** the draft SHALL appear in the draft audit logs list with `status: "pending_review"`
- **THEN** the tool SHALL return `{ draft_id: "<id>" }`

#### Scenario: Agent calls save_draft_audit_log with invalid content
- **WHEN** the agent provides empty or malformed content
- **THEN** the gateway SHALL return 400 and the tool SHALL propagate the error

### Requirement: Draft AuditLog includes per-result reasoning
Each result in the draft AuditLog YAML SHALL include an `agent-reasoning` field explaining the classification.

#### Scenario: Agent classifies a control as Strength
- **WHEN** the agent classifies a result as `Strength`
- **THEN** the result SHALL include `agent-reasoning` referencing specific evidence (count, dates, source match)

#### Scenario: Agent classifies a control as Gap
- **WHEN** the agent classifies a result as `Gap`
- **THEN** the result SHALL include `agent-reasoning` explaining the absence of evidence

### Requirement: Evidence Package artifact
The agent SHALL assemble a factual evidence package before drafting classifications. The evidence package contains raw evidence data mapped to assessment criteria -- no classifications, no judgment.

#### Scenario: Agent assembles evidence
- **WHEN** the agent queries evidence for a target via `query_evidence`
- **THEN** the agent SHALL assemble per-criteria evidence rows
- **THEN** the evidence package SHALL NOT contain classifications or recommendations

### Requirement: Human promotes draft to official record
The gateway SHALL expose `POST /api/audit-logs/promote` requiring an authenticated admin session. The promoting user's identity becomes `created_by` on the official AuditLog.

#### Scenario: Admin promotes a draft
- **WHEN** an admin user POSTs to `/api/audit-logs/promote` with `{ draft_id }`
- **THEN** the gateway SHALL copy the draft to `audit_logs` with `created_by` set to the session user
- **THEN** the draft status SHALL change to `"promoted"`

#### Scenario: Non-admin attempts promotion
- **WHEN** a non-admin user POSTs to `/api/audit-logs/promote`
- **THEN** the gateway SHALL return `403 admin role required`

#### Scenario: Draft already promoted
- **WHEN** a user attempts to promote a draft with `status: "promoted"`
- **THEN** the gateway SHALL return `409 draft already promoted`
