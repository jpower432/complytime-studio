## ADDED Requirements

### Requirement: Posture view displays time-preset filter buttons
The posture view SHALL render a row of time-preset buttons: 7d, 30d, 90d, and All. Clicking a preset SHALL set `selectedTimeRange` to the computed date range relative to the current date and re-fetch posture data. "All" SHALL clear the time range filter.

#### Scenario: Clicking 30d filters to last 30 days
- **WHEN** the user clicks the "30d" preset button
- **THEN** `selectedTimeRange` is set to start=today-30d, end=today, and the posture API is called with those parameters

#### Scenario: Clicking All clears the time filter
- **WHEN** the user clicks the "All" preset button
- **THEN** `selectedTimeRange` is set to null and the posture API is called without start/end parameters

#### Scenario: Active preset is visually indicated
- **WHEN** the "7d" preset is active
- **THEN** the "7d" button has a distinct visual state (accent color) and other presets have the default state

### Requirement: Posture API accepts optional time-range parameters
`GET /api/posture` SHALL accept optional `start` and `end` query parameters. When provided, the response SHALL include only evidence collected within the specified range. When omitted, all evidence is included (current behavior preserved).

#### Scenario: Filtered request returns scoped counts
- **WHEN** `GET /api/posture?start=2026-04-01&end=2026-04-26` is called
- **THEN** the response includes only evidence with `collected_at` between those dates in the pass/fail/other counts

#### Scenario: Unfiltered request returns all evidence
- **WHEN** `GET /api/posture` is called without start/end parameters
- **THEN** the response includes all evidence (backward compatible)

#### Scenario: Invalid date parameters return 400
- **WHEN** `GET /api/posture?start=not-a-date` is called
- **THEN** the API returns HTTP 400 with an error message

### Requirement: PostureStore ListPosture accepts time range
`ListPosture` SHALL accept optional start and end time parameters. When non-zero, the ClickHouse query SHALL filter `collected_at` within the range. When zero, no time filter is applied.

#### Scenario: Time-filtered query restricts evidence window
- **WHEN** `ListPosture(ctx, start=2026-04-01, end=2026-04-26)` is called
- **THEN** the SQL query includes `AND e.collected_at >= '2026-04-01' AND e.collected_at <= '2026-04-26'`

#### Scenario: Zero-time parameters produce unfiltered query
- **WHEN** `ListPosture(ctx, start=zero, end=zero)` is called
- **THEN** the SQL query has no `collected_at` filter (identical to current behavior)
