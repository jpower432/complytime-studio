## ADDED Requirements

### Requirement: Inventory target items are clickable
Each target item in the inventory list SHALL be clickable. Clicking a target SHALL switch to the Evidence tab with a filter chip scoped to that target.

#### Scenario: Click a target
- **WHEN** the user clicks the `web-api` target in the inventory list
- **THEN** the view SHALL switch to the Evidence tab with an active chip `Target: web-api ✕`

#### Scenario: Click a different target after already filtered
- **WHEN** the user is on the Evidence tab with chip `Target: web-api ✕` and navigates back to Inventory and clicks `worker`
- **THEN** the Evidence tab SHALL replace the target chip with `Target: worker ✕`

### Requirement: Inventory control items are clickable
Each control item in the inventory list SHALL be clickable. Clicking a control SHALL switch to the Evidence tab with a filter chip scoped to that control.

#### Scenario: Click a control
- **WHEN** the user clicks `AC-2` in the controls inventory list
- **THEN** the view SHALL switch to the Evidence tab with an active chip `Control: AC-2 ✕`
