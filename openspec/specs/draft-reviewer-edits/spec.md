### Requirement: PATCH endpoint persists reviewer edits
The gateway SHALL expose `PATCH /api/draft-audit-logs/{id}` accepting `{ reviewer_edits: Record<string, { type_override?: string, note?: string }> }`. The endpoint SHALL validate the draft exists and has status `pending_review`. Notes SHALL be truncated to 2000 characters.

#### Scenario: Save type override and note
- **WHEN** the reviewer PATCHes draft `d-123` with `{ reviewer_edits: { "ac-1": { type_override: "Strength", note: "Verified manually" } } }`
- **THEN** the server stores the edits in the `reviewer_edits` column
- **THEN** the response is `200 OK` with `{ status: "saved" }`

#### Scenario: PATCH on promoted draft
- **WHEN** the reviewer PATCHes a draft that has already been promoted
- **THEN** the server returns `409 Conflict` with `{ error: "draft already promoted" }`

#### Scenario: PATCH on non-existent draft
- **WHEN** the reviewer PATCHes a draft ID that does not exist
- **THEN** the server returns `404 Not Found`

### Requirement: Promote merges reviewer edits into official content
When `PromoteDraftAuditLog` is called, the system SHALL read `reviewer_edits` from the draft row, apply type overrides and append notes to the corresponding results in the YAML content, and insert the merged content into the official `audit_logs` table.

#### Scenario: Promote with overrides
- **WHEN** draft `d-123` has `reviewer_edits: { "ac-1": { type_override: "Strength" } }` and original content has result `ac-1` as type `Finding`
- **THEN** the promoted audit log content contains result `ac-1` with type `Strength`

#### Scenario: Promote with note
- **WHEN** draft `d-123` has `reviewer_edits: { "ac-1": { note: "Verified manually" } }`
- **THEN** the promoted audit log content contains result `ac-1` with a `reviewer-note` field set to "Verified manually"

#### Scenario: Promote with no edits
- **WHEN** draft `d-123` has `reviewer_edits: {}` (empty)
- **THEN** the promoted audit log content is identical to the original agent content

### Requirement: Schema includes reviewer_edits column
The `draft_audit_logs` table SHALL include a `reviewer_edits String DEFAULT '{}'` column. Existing rows SHALL default to empty JSON object.

#### Scenario: Column exists after schema init
- **WHEN** the ClickHouse schema is initialized
- **THEN** the `draft_audit_logs` table has a `reviewer_edits` column with default `'{}'`

### Requirement: GET draft returns reviewer_edits
`GET /api/draft-audit-logs/{id}` SHALL include the `reviewer_edits` field in the response JSON.

#### Scenario: Fetch draft with edits
- **WHEN** the reviewer fetches draft `d-123` which has stored edits
- **THEN** the response includes `reviewer_edits` with the stored JSON object
