## MODIFIED Requirements

### Requirement: Job dialog displays available agents
The job creation dialog SHALL fetch the agent directory from `/api/agents` and display each agent as a selectable card showing name, description, skill tags, and the model provider/name backing the agent.

#### Scenario: Agents available
- **WHEN** user opens the New Job dialog and `/api/agents` returns one or more agents
- **THEN** the dialog displays a selectable list of agent cards with name, description, skill tags, and a model badge (e.g., "Claude Sonnet 4" or "Gemini 2.5 Flash")
- **THEN** the first agent is pre-selected

#### Scenario: No agents configured
- **WHEN** user opens the New Job dialog and `/api/agents` returns an empty array
- **THEN** the dialog displays the text input without an agent picker
- **THEN** the job defaults to `studio-threat-modeler`

#### Scenario: Agent fetch fails
- **WHEN** user opens the New Job dialog and `/api/agents` returns an error
- **THEN** the dialog displays the text input without an agent picker
- **THEN** the job defaults to `studio-threat-modeler`

#### Scenario: Model info missing from agent entry
- **WHEN** an agent entry in the `/api/agents` response lacks a `model` field
- **THEN** the agent card omits the model badge
- **THEN** all other card content (name, description, tags) still renders
