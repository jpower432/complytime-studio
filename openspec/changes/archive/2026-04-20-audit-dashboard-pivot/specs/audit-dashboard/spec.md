## ADDED Requirements

### Requirement: Posture summary view
The system SHALL display a compliance posture summary as the default landing page showing pass/fail/gap counts per stored policy, evidence freshness indicators, and trend sparklines across audit periods.

#### Scenario: Dashboard with active policies and evidence
- **WHEN** the user navigates to the Posture view and at least one policy and evidence records exist
- **THEN** the system displays a card per policy with aggregated counts (strengths, findings, gaps, observations) from the most recent AuditLog and a sparkline showing the trend over the last 4 audit periods

#### Scenario: Dashboard with no data
- **WHEN** the user navigates to the Posture view and no policies are stored
- **THEN** the system displays an empty state prompting the user to import a policy from an OCI registry

### Requirement: Evidence freshness indicator
The system SHALL display a freshness indicator per policy showing the time elapsed since the most recent evidence was ingested for that policy's targets.

#### Scenario: Stale evidence
- **WHEN** the most recent evidence for a policy target is older than the policy's assessment frequency interval
- **THEN** the freshness indicator displays a warning state with the elapsed time

### Requirement: Coverage matrix view
The system SHALL display a coverage matrix per MappingDocument showing internal criteria mapped to external framework entries with coverage status (Covered, Partially Covered, Weakly Covered, Not Covered, Unmapped).

#### Scenario: Framework coverage drill-down
- **WHEN** the user selects a MappingDocument from the Policies view
- **THEN** the system displays a matrix with external framework entries as rows and coverage status derived from the most recent AuditLog
