## ADDED Requirements

### Requirement: Job dialog displays available agents
The job creation dialog SHALL fetch the agent directory from `/api/agents` and display each agent as a selectable card showing name, description, and skill tags.

#### Scenario: Agents available
- **WHEN** user opens the New Job dialog and `/api/agents` returns one or more agents
- **THEN** the dialog displays a selectable list of agent cards with name, description, and skill tags
- **THEN** the first agent is pre-selected

#### Scenario: No agents configured
- **WHEN** user opens the New Job dialog and `/api/agents` returns an empty array
- **THEN** the dialog displays the text input without an agent picker
- **THEN** the job defaults to `studio-threat-modeler`

#### Scenario: Agent fetch fails
- **WHEN** user opens the New Job dialog and `/api/agents` returns an error
- **THEN** the dialog displays the text input without an agent picker
- **THEN** the job defaults to `studio-threat-modeler`

### Requirement: Selected agent is bound to the job
The selected agent name SHALL be stored in the job record and used for all A2A communication for that job.

#### Scenario: Agent persists across session
- **WHEN** user selects `studio-gap-analyst` and starts a job
- **THEN** the job store records `agentName: "studio-gap-analyst"`
- **THEN** `sendMessage`, `sendReply`, and `streamTask` route to `/api/a2a/studio-gap-analyst`

#### Scenario: Existing jobs without agentName
- **WHEN** a job loaded from localStorage has no `agentName` field
- **THEN** the system defaults to `studio-threat-modeler`

### Requirement: Job dialog errors are visible
The job creation dialog SHALL display error messages when job creation fails.

#### Scenario: A2A endpoint unreachable
- **WHEN** user clicks "Start Job" and the A2A request fails
- **THEN** the dialog displays the error message in the error area
- **THEN** the "Start Job" button re-enables

#### Scenario: Empty input rejected
- **WHEN** user clicks "Start Job" with an empty text field
- **THEN** the dialog displays "Describe what you want the agent to do."
- **THEN** no network request is made

### Requirement: Single job creation entry point
The jobs view SHALL display exactly one "New Job" button regardless of whether the job list is empty.

#### Scenario: Empty job list
- **WHEN** the job list has zero entries
- **THEN** exactly one "+ New Job" button is visible (in the header)
- **THEN** an empty-state message is shown without a duplicate button

#### Scenario: Non-empty job list
- **WHEN** the job list has one or more entries
- **THEN** exactly one "+ New Job" button is visible (in the header)
