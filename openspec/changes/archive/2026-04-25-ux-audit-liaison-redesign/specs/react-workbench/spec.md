## MODIFIED Requirements

### Requirement: Sidebar navigation items
The sidebar SHALL display the following navigation items in order: Posture, Policies, Evidence, Inbox (with unread badge). The sidebar SHALL NOT include standalone "Audit History" or "Review" items.

#### Scenario: Sidebar shows four items
- **WHEN** the workbench renders
- **THEN** the sidebar displays exactly four nav items: Posture, Policies, Evidence, Inbox

#### Scenario: Inbox badge shows unread count
- **WHEN** the inbox has 3 unread notifications
- **THEN** the Inbox nav item displays a badge with "3"

### Requirement: View routing supports nested paths
The router SHALL support nested hash paths for policy detail drill-down: `#/posture/{policy_id}?tab=requirements|evidence|history`. The `View` type SHALL include `"posture-detail"` as a valid view.

#### Scenario: Nested posture route renders policy detail
- **WHEN** the URL hash is `#/posture/ampel-branch-protection?tab=requirements`
- **THEN** the app renders `PolicyDetailView` with the Requirements tab active

#### Scenario: Legacy audit-history route redirects
- **WHEN** the URL hash is `#/audit-history`
- **THEN** the app redirects to `#/posture`

## REMOVED Requirements

### Requirement: Standalone audit history navigation
**Reason:** Audit History is now a tab within the Posture drill-down, not a standalone view.
**Migration:** `#/audit-history?policy=X` redirects to `#/posture/X?tab=history`.

### Requirement: Standalone draft review navigation
**Reason:** Replaced by the Inbox view which unifies drafts with notifications.
**Migration:** Draft review functionality is accessible via the Inbox view.
