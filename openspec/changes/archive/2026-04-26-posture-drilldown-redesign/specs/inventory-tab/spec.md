## ADDED Requirements

### Requirement: Inventory tab exists in policy detail view
The policy detail view SHALL include an "Inventory" tab between "Requirements" and "Evidence" in the tab bar.

#### Scenario: Tab renders when selected
- **WHEN** a user navigates to `#posture/{id}` and clicks "Inventory"
- **THEN** the Inventory tab SHALL display target and control breakdowns

### Requirement: Target inventory with posture bars
The Inventory tab SHALL display a list of distinct targets derived from evidence rows for the current policy. Each target entry SHALL show target name (or target_id as fallback), total evidence count, and a mini posture bar showing pass/fail/other proportions.

#### Scenario: Policy with evidence across 3 targets
- **WHEN** the policy has evidence rows for targets `cluster-a` (30 pass, 8 fail), `cluster-b` (18 pass, 10 fail), `node-pool` (15 pass, 0 fail)
- **THEN** the Inventory tab SHALL list 3 targets, each with a posture bar reflecting their individual pass/fail ratio

#### Scenario: No evidence for policy
- **WHEN** the policy has zero evidence rows
- **THEN** the Inventory tab SHALL display an empty state message

### Requirement: Control inventory with pass rate
The Inventory tab SHALL display a list of distinct controls derived from evidence rows for the current policy. Each control entry SHALL show the control_id, evidence count, and pass rate percentage.

#### Scenario: Control with mixed results
- **WHEN** control `AC-1` has 20 evidence rows (15 Passed, 5 Failed)
- **THEN** the control entry SHALL show `AC-1`, `20 records`, `75% pass`

### Requirement: Inventory data sourced from existing evidence endpoint
The Inventory tab SHALL compute all aggregations client-side from evidence rows returned by `GET /api/evidence?policy_id={id}`. No new backend endpoint is required.

#### Scenario: Client-side grouping
- **WHEN** the evidence API returns 100 rows for the policy
- **THEN** the Inventory tab SHALL GROUP BY `target_id` and `control_id` to produce the inventory lists without additional API calls
