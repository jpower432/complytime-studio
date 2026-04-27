## ADDED Requirements

### Requirement: FilterChip component renders active filters as dismissible chips
The system SHALL render each active filter as a chip displaying `"Label: value"` with a dismiss button (`✕`). Clicking `✕` SHALL remove the filter and re-execute the query.

#### Scenario: Single active filter displayed
- **WHEN** a filter is active with field `Target` and value `web-api`
- **THEN** a chip SHALL render displaying `Target: web-api ✕` above the data table

#### Scenario: Dismiss a filter chip
- **WHEN** the user clicks `✕` on an active filter chip
- **THEN** the chip SHALL be removed, the filter cleared, and the table re-filtered

### Requirement: Multiple filter chips combine with AND logic
The system SHALL support multiple active filter chips simultaneously. All active filters SHALL combine with AND logic when filtering the data set.

#### Scenario: Two active filters
- **WHEN** chips `Target: web-api` and `Result: Failed` are both active
- **THEN** the table SHALL display only rows where target is `web-api` AND result is `Failed`

### Requirement: Filter chips sync with filter controls
The system SHALL keep filter chips and their corresponding filter controls in sync. Clearing a chip SHALL reset the corresponding control. Setting a control SHALL create the corresponding chip.

#### Scenario: Chip cleared resets dropdown
- **WHEN** the user dismisses the `Result: Failed` chip
- **THEN** the Result filter control SHALL reset to its default unselected state

### Requirement: Cross-view navigation creates filter chips
The system SHALL create a filter chip when a user navigates from one view to another with a scoped filter (e.g., clicking an inventory item to open the evidence tab).

#### Scenario: Inventory target click creates chip on evidence tab
- **WHEN** the user clicks a target item in the inventory tab
- **THEN** the system SHALL switch to the evidence tab with a `Target: <target_name> ✕` chip active
