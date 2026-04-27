## MODIFIED Requirements

### Requirement: Posture card navigates to policy detail view
The system SHALL navigate to a policy detail view when the user clicks a posture card's "View Details" button. The detail view SHALL display the policy name as a breadcrumb: Posture > [Policy Title]. The posture card itself SHALL NOT be a full clickable surface — the stacked bar and recency border are non-interactive visual elements; only the "View Details" button triggers navigation.

#### Scenario: Click posture card button opens policy detail
- **WHEN** the user clicks the "View Details" button on a posture card for "ampel-branch-protection"
- **THEN** the URL updates to `#/posture/ampel-branch-protection` and the policy detail view renders with breadcrumb "Posture > Ampel Branch Protection"

#### Scenario: Breadcrumb navigates back to posture
- **WHEN** the user clicks "Posture" in the breadcrumb
- **THEN** the view returns to the posture grid showing all policy cards
