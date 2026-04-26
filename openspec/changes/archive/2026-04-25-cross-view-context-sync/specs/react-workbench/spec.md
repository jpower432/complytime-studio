## ADDED Requirements

### Requirement: All policy selectors write selectedPolicyId on search
Every view with a policy dropdown SHALL write `selectedPolicyId` when the user triggers a search or applies filters.

#### Scenario: Evidence view sets policy signal
- **WHEN** the user selects a policy in evidence view and clicks Search
- **THEN** `selectedPolicyId` is updated to the selected value

#### Scenario: Audit history sets policy signal
- **WHEN** the user selects a policy in audit history and clicks Search
- **THEN** `selectedPolicyId` is updated to the selected value

### Requirement: Date range filters write selectedTimeRange
Views with start/end date inputs SHALL write `selectedTimeRange` when the user triggers a search.

#### Scenario: Audit history sets time range
- **WHEN** the user sets start and end dates in audit history and clicks Search
- **THEN** `selectedTimeRange` is updated with `{ start, end }`

#### Scenario: Requirement matrix sets time range
- **WHEN** the user sets audit start/end in requirement matrix and clicks Search
- **THEN** `selectedTimeRange` is updated with `{ start, end }`

### Requirement: Control family filter writes selectedControlId
The requirement matrix SHALL write `selectedControlId` when a control family filter is applied.

#### Scenario: Filter by control family
- **WHEN** the user selects control family "AC" in the requirement matrix and clicks Search
- **THEN** `selectedControlId` is updated to "AC"

#### Scenario: Clear control family filter
- **WHEN** the user selects "All control families" and clicks Search
- **THEN** `selectedControlId` is set to null

### Requirement: Views pre-fill filters from shared signals on mount
Each view SHALL read shared signals on mount and pre-fill local filter state if the signal is non-null.

#### Scenario: Navigate from posture to evidence
- **WHEN** the user clicks a posture card (setting `selectedPolicyId`) then navigates to evidence
- **THEN** the evidence view policy dropdown is pre-filled with the selected policy

#### Scenario: Navigate from audit history to requirements
- **WHEN** the user searches audit history with dates (setting `selectedTimeRange`) then navigates to requirements
- **THEN** the requirement matrix date inputs are pre-filled

### Requirement: Requirement matrix refetches on viewInvalidation
The requirement matrix SHALL refetch data when `viewInvalidation` changes, provided a policy is selected.

#### Scenario: Agent produces artifact
- **WHEN** the agent produces an AuditLog artifact that triggers `invalidateViews()`
- **THEN** the requirement matrix refetches if a policy is currently selected
