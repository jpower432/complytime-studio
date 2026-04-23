## ADDED Requirements

### Requirement: Detect AuditLog YAML in finalized agent messages
The chat UI SHALL scan finalized agent text messages for fenced YAML code blocks containing Gemara AuditLog artifacts.

#### Scenario: Agent response contains AuditLog YAML
- **WHEN** the agent stream completes and the finalized text contains a fenced `yaml` code block
- **THEN** the system SHALL parse each YAML code block
- **THEN** if `metadata.type` equals `"AuditLog"`, the block SHALL be extracted as a detected artifact

#### Scenario: Agent response contains no YAML
- **WHEN** the agent stream completes and the finalized text contains no fenced YAML blocks
- **THEN** no artifact detection occurs and the message renders normally

#### Scenario: YAML block is not an AuditLog
- **WHEN** a fenced YAML block parses successfully but `metadata.type` is not `"AuditLog"`
- **THEN** the block SHALL NOT be extracted as an artifact
- **THEN** the block SHALL render as a normal code block in the message

#### Scenario: YAML block fails to parse
- **WHEN** a fenced YAML block cannot be parsed as valid YAML
- **THEN** the block SHALL NOT be extracted as an artifact
- **THEN** the block SHALL render as a normal code block in the message

### Requirement: Render detected AuditLogs as artifact cards
Detected AuditLog artifacts SHALL be rendered using the existing artifact card component with a "Save to Audit History" button.

#### Scenario: Single AuditLog detected
- **WHEN** one AuditLog is detected in a finalized message
- **THEN** the text portions of the message SHALL render as a normal message
- **THEN** the AuditLog SHALL render as an artifact card after the text message
- **THEN** the artifact card SHALL display the artifact name derived from `metadata.id`
- **THEN** the artifact card SHALL show a preview of the YAML content
- **THEN** the artifact card SHALL include a "Save to Audit History" button for admin users

#### Scenario: Multiple AuditLogs detected
- **WHEN** multiple AuditLogs are detected in a finalized message
- **THEN** each AuditLog SHALL render as a separate artifact card
- **THEN** each card SHALL have its own "Save to Audit History" button

### Requirement: Save button persists AuditLog to ClickHouse
The "Save to Audit History" button SHALL POST the AuditLog YAML to `POST /api/audit-logs` using the existing `saveAuditLog()` function.

#### Scenario: User saves AuditLog
- **WHEN** an admin user clicks "Save to Audit History" on a detected artifact card
- **THEN** the system SHALL POST the YAML content to `/api/audit-logs`
- **THEN** on success, a confirmation message SHALL appear in the chat

#### Scenario: Save fails
- **WHEN** the POST to `/api/audit-logs` returns a non-200 status
- **THEN** an error message SHALL appear in the chat with the failure reason

#### Scenario: Non-admin user
- **WHEN** a viewer-role user sees a detected artifact card
- **THEN** the "Save to Audit History" button SHALL NOT be visible
