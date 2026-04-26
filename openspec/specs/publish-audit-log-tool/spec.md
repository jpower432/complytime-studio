### Requirement: Agent publishes draft AuditLog via custom tool
The agent SHALL have a `publish_audit_log` function tool that persists a validated AuditLog YAML as a **draft** via the internal Gateway endpoint and emits it as an ADK artifact for frontend preview.

#### Scenario: Agent calls publish_audit_log with valid AuditLog
- **WHEN** the agent calls `publish_audit_log(yaml_content=<valid AuditLog YAML>)`
- **THEN** the tool SHALL parse the YAML and verify `metadata.type` equals `"AuditLog"`
- **THEN** the tool SHALL POST to `/internal/draft-audit-logs` (no auth required, cluster-internal only)
- **THEN** the tool SHALL call `ToolContext.save_artifact()` for frontend preview
- **THEN** the draft SHALL appear in the workbench review queue

#### Scenario: Agent calls publish_audit_log with invalid YAML
- **WHEN** the agent calls `publish_audit_log(yaml_content=<invalid YAML>)`
- **THEN** the tool SHALL return `{"error": "Invalid YAML: ..."}` without persisting

#### Scenario: Agent calls publish_audit_log with non-AuditLog type
- **WHEN** the agent calls `publish_audit_log` with YAML where `metadata.type` is not `"AuditLog"`
- **THEN** the tool SHALL return `{"error": "Expected metadata.type=AuditLog, got '...'"}`

### Requirement: Draft AuditLog includes per-result reasoning
Each result in the draft AuditLog YAML SHALL include an `agent-reasoning` field explaining the classification.

#### Scenario: Agent classifies a control as Strength
- **WHEN** the agent classifies a result as `Strength`
- **THEN** the result SHALL include `agent-reasoning` referencing specific evidence (count, dates, source match)

#### Scenario: Agent classifies a control as Gap
- **WHEN** the agent classifies a result as `Gap`
- **THEN** the result SHALL include `agent-reasoning` explaining the absence of evidence

### Requirement: Evidence Package artifact
The agent SHALL emit a factual evidence package as a separate artifact before drafting classifications. The evidence package contains raw evidence data mapped to assessment criteria — no classifications, no judgment.

#### Scenario: Agent assembles evidence
- **WHEN** the agent queries evidence for a target
- **THEN** the agent SHALL emit an evidence package artifact (application/yaml) with per-criteria evidence rows
- **THEN** the evidence package SHALL NOT contain classifications or recommendations

### Requirement: Internal draft endpoint
The Gateway SHALL expose `POST /internal/draft-audit-logs` on the cluster-internal interface. This endpoint requires no authentication. Access restricted by NetworkPolicy.

#### Scenario: Agent persists a draft
- **WHEN** the agent POSTs a draft AuditLog to `/internal/draft-audit-logs`
- **THEN** the Gateway SHALL INSERT into `draft_audit_logs` with `status: "pending_review"`
- **THEN** the Gateway SHALL return `{"status": "drafted", "draft_id": "<id>"}`

### Requirement: Human promotes draft to official record
The Gateway SHALL expose `POST /api/audit-logs/promote` requiring an authenticated admin session. The promoting user's identity becomes `created_by` on the official AuditLog.

#### Scenario: Admin promotes a draft
- **WHEN** an admin user POSTs to `/api/audit-logs/promote` with `{draft_id, overrides?}`
- **THEN** the Gateway SHALL copy the draft to `audit_logs` with `created_by` set to the session user
- **THEN** the draft status SHALL change to `"promoted"`
- **THEN** any `overrides` SHALL be applied to the result classifications before promotion

#### Scenario: Non-admin attempts promotion
- **WHEN** a non-admin user POSTs to `/api/audit-logs/promote`
- **THEN** the Gateway SHALL return `403 admin role required`

#### Scenario: Draft already promoted
- **WHEN** a user attempts to promote a draft with `status: "promoted"`
- **THEN** the Gateway SHALL return `409 draft already promoted`

