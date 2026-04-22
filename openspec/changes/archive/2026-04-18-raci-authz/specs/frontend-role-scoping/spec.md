## ADDED Requirements

### Requirement: Workbench fetches access set on load
The workbench SHALL fetch `GET /auth/me` on initial load and store the `policies` access map for use in conditional rendering.

#### Scenario: Access map available
- **WHEN** the workbench loads and `/auth/me` returns a `policies` map
- **THEN** the access map is stored in application state and available to all components

### Requirement: Role-aware policy list
The policy list view SHALL display a RACI role badge next to each policy name.

#### Scenario: Policy with responsible role
- **WHEN** the user has `responsible` role for a policy
- **THEN** a badge showing "Responsible" is displayed next to the policy name

#### Scenario: Policy with informed role
- **WHEN** the user has `informed` role for a policy
- **THEN** a badge showing "Informed" is displayed next to the policy name

### Requirement: Write controls hidden for read-only roles
Import and upload buttons SHALL be hidden when the user's highest role for the current policy context is `consulted` or `informed`.

#### Scenario: Informed user views policy detail
- **WHEN** a user with `informed` role views a policy's detail page
- **THEN** the "Import Mapping" and "Upload Evidence" buttons are not rendered

#### Scenario: Responsible user views policy detail
- **WHEN** a user with `responsible` role views a policy's detail page
- **THEN** the "Import Mapping" and "Upload Evidence" buttons are visible

### Requirement: Audit prompts hidden for informed users
The chat assistant SHALL hide "Run audit" style prompts for users with `informed` role on the active policy.

#### Scenario: Informed user in chat
- **WHEN** a user with `informed` role interacts with the chat assistant
- **THEN** audit trigger prompts are not displayed in the suggested actions

### Requirement: Empty state when no policies visible
When the user has no policies in their access set (and enforcement is active), the workbench SHALL show guidance.

#### Scenario: Zero visible policies
- **WHEN** the policy list is empty because the user has no RACI access
- **THEN** the workbench displays "No policies visible. Import a policy or contact a policy owner."
