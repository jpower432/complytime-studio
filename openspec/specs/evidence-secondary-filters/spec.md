### Requirement: Add Filter menu lists available secondary filter fields
The system SHALL display an "+ Filter" button in the evidence filter bar. Clicking it SHALL open a menu listing available secondary filter fields: Target, Result, Engine, Compliance Status, Owner, Enrichment Status.

#### Scenario: Open Add Filter menu
- **WHEN** the user clicks "+ Filter"
- **THEN** a dropdown menu SHALL appear listing all available secondary filter fields

#### Scenario: Active filter excluded from menu
- **WHEN** the user already has an active `Result: Failed` chip
- **THEN** "Result" SHALL NOT appear in the "+ Filter" menu

### Requirement: Selecting a field shows a value selector
The system SHALL display an inline value selector after the user picks a field from the "+ Filter" menu. Enum fields (Result, Compliance Status) SHALL show fixed options. Dynamic fields (Target, Engine, Owner, Enrichment Status) SHALL populate values from the current data set.

#### Scenario: Select Result field
- **WHEN** the user selects "Result" from the "+ Filter" menu
- **THEN** a dropdown SHALL appear with options: Passed, Failed, Unknown

#### Scenario: Select Target field with data
- **WHEN** the user selects "Target" from the "+ Filter" menu and evidence records contain targets `web-api`, `worker`, `db-proxy`
- **THEN** a dropdown SHALL appear populated with `web-api`, `worker`, `db-proxy`

### Requirement: Selecting a value creates a chip and filters
The system SHALL create a filter chip and re-filter the table when the user selects a value from the inline selector. The menu SHALL close after selection.

#### Scenario: Select Failed result
- **WHEN** the user selects "Failed" from the Result value selector
- **THEN** a chip `Result: Failed ✕` SHALL appear, the menu SHALL close, and the table SHALL show only rows with `eval_result = Failed`

### Requirement: Primary filters remain always visible
Policy dropdown, Control ID text input, date range inputs, and Search button SHALL remain always visible in the filter bar. These SHALL NOT be moved into the "+ Filter" menu.

#### Scenario: Page load
- **WHEN** the evidence page loads
- **THEN** the policy dropdown, control ID input, date range inputs, "+ Filter" button, and Search button SHALL be visible
