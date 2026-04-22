## MODIFIED Requirements

### Requirement: Save AuditLog from chat
The `saveAuditLog` function SHALL send only `policy_id` and `content` to `POST /api/audit-logs`. It SHALL NOT send `audit_start`, `audit_end`, or `summary` — the gateway derives these from the YAML content.

#### Scenario: User clicks Save to Audit History
- **WHEN** the user clicks "Save to Audit History" on an artifact card
- **THEN** `saveAuditLog` SHALL POST `{"policy_id": selectedPolicyId, "content": artifact.content}` to `/api/audit-logs`

#### Scenario: Gateway returns parse error
- **WHEN** the gateway returns `400 Bad Request` because the artifact content is invalid AuditLog YAML
- **THEN** the UI SHALL display the error message to the user
