## ADDED Requirements

### Requirement: Posture card displays stacked progress bar
Each posture card SHALL render a horizontal stacked bar showing the ratio of passed, failed, and other evidence counts. The bar SHALL use CSS flex with percentage-based segment widths. The bar SHALL include `role="img"` and an `aria-label` describing the counts.

#### Scenario: Card with mixed results shows proportional segments
- **WHEN** a policy has 7 passed, 2 failed, 1 other (10 total)
- **THEN** the bar renders three segments: 70% green, 20% red, 10% amber

#### Scenario: Card with all passed shows single green segment
- **WHEN** a policy has 5 passed, 0 failed, 0 other
- **THEN** the bar renders one green segment at 100% width

#### Scenario: Card with no evidence shows no bar
- **WHEN** a policy has 0 total evidence rows
- **THEN** no progress bar is rendered

### Requirement: Posture card displays evidence recency via border color
Each posture card SHALL display a left border colored by evidence freshness. Thresholds: current (<=7 days, success color), aging (<=30 days, warning color), stale (>30 days, error color), none (no evidence, gray color).

#### Scenario: Fresh evidence shows success border
- **WHEN** a policy's latest evidence was collected 2 days ago
- **THEN** the card's left border is the success color (green)

#### Scenario: Aging evidence shows warning border
- **WHEN** a policy's latest evidence was collected 15 days ago
- **THEN** the card's left border is the warning color (amber)

#### Scenario: Stale evidence shows error border
- **WHEN** a policy's latest evidence was collected 45 days ago
- **THEN** the card's left border is the error color (red)

#### Scenario: No evidence shows gray border
- **WHEN** a policy has no evidence records
- **THEN** the card's left border is the gray color

### Requirement: Posture view displays aggregate summary strip
The posture view SHALL render a summary strip above the card grid showing: total policy count, overall pass rate (percentage), a full-width stacked bar of aggregate counts, and a stale-evidence warning count when any policies have stale or missing evidence.

#### Scenario: Summary shows aggregate across all policies
- **WHEN** three policies exist with combined 20 passed, 5 failed, 3 other
- **THEN** the summary strip shows "3 policies", "71% overall pass rate", a stacked bar, and no stale warning

#### Scenario: Summary shows stale warning
- **WHEN** two of three policies have latest evidence older than 30 days
- **THEN** the summary strip includes "2 with stale evidence" in the error color

#### Scenario: Summary hidden when no policies exist
- **WHEN** no policies are imported
- **THEN** the summary strip is not rendered (empty state shows instead)
