## MODIFIED Requirements

### Requirement: Remove dead framework filter
The Evidence tab SHALL NOT display a "Framework" filter input. The previous input sent a `framework` query parameter that the backend ignores.

#### Scenario: Filter bar renders without framework
- **WHEN** the Evidence tab loads
- **THEN** the filter bar SHALL contain Policy, Control ID, date range, and Search — no Framework input

### Requirement: Remove target name from "More Filters"
The hidden "More Filters" section and its "Target name" input SHALL be removed. Target filtering is better served by the Inventory tab's target breakdown.

#### Scenario: No "More Filters" toggle
- **WHEN** the Evidence tab loads
- **THEN** there SHALL be no "More Filters" button or expandable filter section

## ADDED Requirements

### Requirement: Evidence summary strip
The Evidence tab SHALL display a summary strip above the table showing: total record count, overall pass rate (percentage), distinct engine count, and distinct target count. Values SHALL be computed client-side from the returned evidence rows.

#### Scenario: Summary strip with data
- **WHEN** the Evidence tab has 142 records, 102 Passed, 3 distinct engines, 12 distinct targets
- **THEN** the summary strip SHALL show `142 records`, `72% pass`, `3 engines`, `12 targets` with an aggregate posture bar

#### Scenario: No evidence
- **WHEN** the Evidence tab has zero records
- **THEN** the summary strip SHALL NOT render

### Requirement: Row-level recency fading
Each evidence row SHALL have a visual treatment based on the age of its `collected_at` timestamp. Rows SHALL progressively fade and display a stale indicator as they age. Thresholds: ≤7d full opacity, 8–30d slightly faded, 31–90d faded with stale badge, >90d very faded with stale badge.

#### Scenario: Recent evidence row
- **WHEN** an evidence row has `collected_at` within the last 7 days
- **THEN** the row SHALL render at full opacity with no stale indicator

#### Scenario: Aging evidence row
- **WHEN** an evidence row has `collected_at` 15 days ago
- **THEN** the row SHALL render at reduced opacity (0.75)

#### Scenario: Stale evidence row
- **WHEN** an evidence row has `collected_at` 60 days ago
- **THEN** the row SHALL render at low opacity (0.5) with a visible stale badge

### Requirement: Shared freshness utility
Freshness thresholds and classification functions SHALL be extracted from `posture-view.tsx` into a shared utility module so that both posture cards and evidence rows use the same constants and logic.

#### Scenario: Consistent thresholds
- **WHEN** `STALE_THRESHOLD_DAYS` is changed in the shared utility
- **THEN** both posture card borders and evidence row fading SHALL reflect the new value
