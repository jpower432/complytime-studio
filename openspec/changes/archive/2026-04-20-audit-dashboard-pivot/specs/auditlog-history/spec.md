## ADDED Requirements

### Requirement: Store AuditLogs
The system SHALL store AuditLog artifacts produced by the gap analyst in ClickHouse with metadata (audit_id, policy_id, audit_start, audit_end, framework, created_at, created_by) and pre-computed summary counts.

#### Scenario: AuditLog created by agent
- **WHEN** the gap analyst agent emits a validated AuditLog artifact
- **THEN** the system stores it in the `audit_logs` ClickHouse table with the full YAML content and a JSON summary of result type counts

### Requirement: Browse audit history
The system SHALL display a timeline of AuditLogs for a given policy, ordered by audit period.

#### Scenario: Audit History view
- **WHEN** the user navigates to Audit History and selects a policy
- **THEN** the system displays AuditLogs as a timeline with summary cards showing result counts per audit period

### Requirement: Compare audit periods
The system SHALL allow users to compare two AuditLog summaries side-by-side to identify posture changes.

#### Scenario: Quarter-over-quarter comparison
- **WHEN** the user selects two audit periods for the same policy
- **THEN** the system displays a side-by-side comparison showing delta in strengths, findings, gaps, and observations with directional indicators (improved/regressed/unchanged)

### Requirement: Drill into AuditLog results
The system SHALL allow users to drill into individual AuditResults within an AuditLog.

#### Scenario: View individual result
- **WHEN** the user selects an AuditResult from an AuditLog
- **THEN** the system displays the result type, criteria reference, evidence summary, and recommendations in a detail panel
