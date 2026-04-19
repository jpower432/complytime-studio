## ADDED Requirements

### Requirement: Mission dialog displays available agents
The mission creation dialog SHALL fetch the agent directory from `/api/agents` and display each agent as a selectable card showing name, description, and skill tags.

#### Scenario: Agents available
- **WHEN** user opens the New Mission dialog and `/api/agents` returns one or more agents
- **THEN** the dialog displays a selectable list of agent cards with name, description, and skill tags
- **THEN** the first agent is pre-selected

#### Scenario: No agents configured
- **WHEN** user opens the New Mission dialog and `/api/agents` returns an empty array
- **THEN** the dialog displays the text input without an agent picker
- **THEN** the mission defaults to `studio-threat-modeler`

#### Scenario: Agent fetch fails
- **WHEN** user opens the New Mission dialog and `/api/agents` returns an error
- **THEN** the dialog displays the text input without an agent picker
- **THEN** the mission defaults to `studio-threat-modeler`

### Requirement: Selected agent is bound to the mission
The selected agent name SHALL be stored in the mission record and used for all A2A communication for that mission.

#### Scenario: Agent persists across session
- **WHEN** user selects `studio-gap-analyst` and starts a mission
- **THEN** the mission store records `agentName: "studio-gap-analyst"`
- **THEN** `sendMessage`, `sendReply`, and `streamTask` route to `/api/a2a/studio-gap-analyst`

#### Scenario: Existing missions without agentName
- **WHEN** a mission loaded from localStorage has no `agentName` field
- **THEN** the system defaults to `studio-threat-modeler`

### Requirement: Mission dialog errors are visible
The mission creation dialog SHALL display error messages when mission creation fails.

#### Scenario: A2A endpoint unreachable
- **WHEN** user clicks "Start Mission" and the A2A request fails
- **THEN** the dialog displays the error message in the error area
- **THEN** the "Start Mission" button re-enables

#### Scenario: Empty input rejected
- **WHEN** user clicks "Start Mission" with an empty text field
- **THEN** the dialog displays "Describe what you want the agent to do."
- **THEN** no network request is made

### Requirement: Single mission creation entry point
The missions view SHALL display exactly one "New Mission" button regardless of whether the mission list is empty.

#### Scenario: Empty mission list
- **WHEN** the mission list has zero entries
- **THEN** exactly one "+ New Mission" button is visible (in the header)
- **THEN** an empty-state message is shown without a duplicate button

#### Scenario: Non-empty mission list
- **WHEN** the mission list has one or more entries
- **THEN** exactly one "+ New Mission" button is visible (in the header)
