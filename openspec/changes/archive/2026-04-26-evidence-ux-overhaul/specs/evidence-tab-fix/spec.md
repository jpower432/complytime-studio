## ADDED Requirements

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

## MODIFIED Requirements

### Requirement: Row-level recency fading
Each evidence row SHALL have a background tint based on its freshness bucket as determined by the frequency-aware staleness model. The previous opacity-based fading is replaced with neutral background tints. The "stale" text badge is removed — the background tint is the signal.

#### Scenario: Recent evidence row
- **WHEN** an evidence row is classified as Current
- **THEN** the row SHALL render with a minimal neutral tint and no stale badge

#### Scenario: Stale evidence row
- **WHEN** an evidence row is classified as Stale
- **THEN** the row SHALL render with a prominent neutral tint and no stale text badge
