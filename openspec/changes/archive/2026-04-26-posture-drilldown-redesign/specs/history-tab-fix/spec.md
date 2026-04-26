## MODIFIED Requirements

### Requirement: Audit history renders as table, not cards
The History tab SHALL render audit logs in a table with columns: Period, Framework, Strengths, Findings, Gaps, Author. The previous card-based layout is replaced.

#### Scenario: Three audit logs
- **WHEN** the policy has 3 audit logs
- **THEN** the History tab SHALL display a 3-row table ordered by `audit_start DESC`

### Requirement: Period-over-period deltas
Each row in the history table SHALL display delta values compared to the next (chronologically previous) row. Deltas SHALL be computed client-side by diffing the parsed `summary` JSON of adjacent rows.

#### Scenario: Findings decreased
- **WHEN** the current audit has 2 findings and the previous audit had 5 findings
- **THEN** the Findings cell SHALL display `2 (−3)` with the delta in a success color

#### Scenario: Gaps increased
- **WHEN** the current audit has 3 gaps and the previous audit had 1 gap
- **THEN** the Gaps cell SHALL display `3 (+2)` with the delta in a warning color

#### Scenario: First (oldest) audit log
- **WHEN** there is no previous audit to compare against
- **THEN** the row SHALL display counts with no delta

### Requirement: Audit ID dropdown when embedded
When the History tab is rendered inside the policy detail view (embedded mode), the Audit ID filter SHALL be a `<select>` dropdown populated from the fetched audit logs. The standalone History view SHALL keep the text input for cross-policy lookup.

#### Scenario: Embedded dropdown
- **WHEN** the History tab is rendered at `#posture/{id}` with 3 audit logs loaded
- **THEN** the Audit ID filter SHALL be a `<select>` with options for each `audit_id` plus an empty "All" option

#### Scenario: Standalone text input
- **WHEN** the History tab is rendered as the standalone Audit History view (not embedded)
- **THEN** the Audit ID filter SHALL remain a text input

### Requirement: Click-to-expand YAML detail
Clicking a history table row SHALL expand an inline detail panel below the row showing the full audit log YAML content and a Download YAML button. This replaces the separate "Audit Detail" panel.

#### Scenario: Expand audit detail
- **WHEN** a user clicks a history table row
- **THEN** an inline panel SHALL expand below the row with a `<pre>` YAML viewer and a "Download YAML" button

#### Scenario: Collapse audit detail
- **WHEN** a user clicks the same row again (or clicks a different row)
- **THEN** the expanded panel SHALL collapse
