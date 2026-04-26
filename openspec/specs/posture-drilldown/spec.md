## Requirements

### Requirement: Posture card navigates to policy detail view
The system SHALL navigate to a policy detail view when the user clicks a posture card. The detail view SHALL display the policy name as a breadcrumb: Posture > [Policy Title].

#### Scenario: Click posture card opens policy detail
- **WHEN** the user clicks a posture card for "ampel-branch-protection"
- **THEN** the URL updates to `#/posture/ampel-branch-protection` and the policy detail view renders with breadcrumb "Posture > Ampel Branch Protection"

#### Scenario: Breadcrumb navigates back to posture
- **WHEN** the user clicks "Posture" in the breadcrumb
- **THEN** the view returns to the posture grid showing all policy cards

### Requirement: Policy detail view has tabbed layout
The system SHALL display three tabs within the policy detail view: Requirements, Evidence, and History. The active tab SHALL be reflected in the URL hash as `?tab=requirements|evidence|history`.

#### Scenario: Default tab is Requirements
- **WHEN** the user opens a policy detail view without a tab parameter
- **THEN** the Requirements tab is active and the requirement matrix for that policy loads

#### Scenario: Tab selection persists in URL
- **WHEN** the user clicks the "History" tab
- **THEN** the URL updates to `#/posture/{policy_id}?tab=history` and the audit history for that policy loads

### Requirement: Audit History tab shows per-policy audit logs
The system SHALL display the same audit history functionality (log list, detail, comparison, YAML download) within the History tab, scoped to the active policy. No standalone Audit History nav item SHALL exist.

#### Scenario: Audit logs load for active policy
- **WHEN** the user opens the History tab for policy "ampel-branch-protection"
- **THEN** audit logs for that policy load without requiring a policy selector dropdown

### Requirement: Legacy audit-history hash redirects
The system SHALL redirect `#/audit-history` to `#/posture` for backward compatibility with existing bookmarks and deep links.

#### Scenario: Old deep link redirects
- **WHEN** the user navigates to `#/audit-history?policy=ampel-branch-protection`
- **THEN** the URL redirects to `#/posture/ampel-branch-protection?tab=history`
