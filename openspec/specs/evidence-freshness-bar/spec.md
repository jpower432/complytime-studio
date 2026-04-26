### Requirement: Freshness bar visualizes staleness distribution
The system SHALL render an interactive segmented bar above the evidence table showing the proportion of evidence records in each freshness bucket (Current, Aging, Stale, Very Stale). No counts or percentages SHALL be displayed — the segment proportions are the data.

#### Scenario: Mixed freshness distribution
- **WHEN** evidence records span multiple freshness buckets
- **THEN** the bar SHALL display four contiguous segments whose widths are proportional to the record count in each bucket

#### Scenario: All evidence current
- **WHEN** all evidence records fall within the Current bucket
- **THEN** the bar SHALL display a single full-width Current segment

### Requirement: Freshness bar segments show tooltip on hover
Each segment of the freshness bar SHALL display a tooltip on hover identifying the bucket name (e.g., "Current", "Stale").

#### Scenario: Hover over stale segment
- **WHEN** the user hovers over the Stale segment of the freshness bar
- **THEN** a tooltip SHALL display "Stale"

### Requirement: Clicking a freshness bar segment creates a filter chip
Clicking a segment of the freshness bar SHALL create a `Freshness: <bucket>` filter chip, filtering the evidence table to show only rows in that bucket.

#### Scenario: Click stale segment
- **WHEN** the user clicks the Stale segment of the freshness bar
- **THEN** a chip `Freshness: Stale ✕` SHALL appear and the table SHALL show only stale evidence rows

#### Scenario: Clear freshness chip
- **WHEN** the user dismisses the `Freshness: Stale ✕` chip
- **THEN** the table SHALL return to showing all evidence rows and the freshness bar SHALL remain visible

### Requirement: Freshness bar uses neutral mode-adaptive colors
The freshness bar segments SHALL use neutral tint shades that render well in both light and dark mode. No primary colors (red, green, yellow, orange) SHALL be used.

#### Scenario: Dark mode rendering
- **WHEN** the user has dark mode enabled
- **THEN** the freshness bar segments SHALL use dark-mode neutral tints with sufficient contrast between segments
