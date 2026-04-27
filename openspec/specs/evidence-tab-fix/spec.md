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
Each evidence row SHALL have a background tint based on its freshness bucket as determined by the frequency-aware staleness model. The previous opacity-based fading is replaced with neutral background tints. The "stale" text badge is removed — the background tint is the signal.

#### Scenario: Recent evidence row
- **WHEN** an evidence row is classified as Current
- **THEN** the row SHALL render with a minimal neutral tint and no stale badge

#### Scenario: Stale evidence row
- **WHEN** an evidence row is classified as Stale
- **THEN** the row SHALL render with a prominent neutral tint and no stale text badge

### Requirement: Shared freshness utility
Freshness thresholds and classification functions SHALL be extracted from `posture-view.tsx` into a shared utility module so that both posture cards and evidence rows use the same constants and logic.

#### Scenario: Consistent thresholds
- **WHEN** `STALE_THRESHOLD_DAYS` is changed in the shared utility
- **THEN** both posture card borders and evidence row fading SHALL reflect the new value

### Requirement: Evidence tab hides upload controls when embedded
The Evidence tab SHALL NOT display the "Upload Evidence" button or manual entry form when rendered inside the policy detail view (embedded mode). Upload and manual entry SHALL only be available on the main Evidence page.

#### Scenario: Embedded evidence tab
- **WHEN** the Evidence tab is rendered at `#posture/{id}?tab=evidence`
- **THEN** the "Upload Evidence" button and manual entry form SHALL NOT be visible, regardless of user role

#### Scenario: Main evidence page
- **WHEN** the Evidence page is rendered from the sidebar navigation
- **THEN** the "Upload Evidence" button SHALL be visible for admin users

### Requirement: Evidence rows use neutral background tint for freshness
Each evidence row SHALL have a background tint based on its freshness bucket. Tints SHALL use neutral HSL shades at low opacity (~8%), mode-adaptive via CSS custom properties. No primary colors.

#### Scenario: Current evidence row in light mode
- **WHEN** an evidence row is classified as Current and light mode is active
- **THEN** the row background SHALL use a soft gray-blue tint, barely visible

#### Scenario: Very stale evidence row in dark mode
- **WHEN** an evidence row is classified as Very Stale and dark mode is active
- **THEN** the row background SHALL use a near-white gray tint, prominent against the dark background
