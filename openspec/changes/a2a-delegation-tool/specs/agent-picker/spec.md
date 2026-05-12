## MODIFIED Requirements

### Requirement: Job dialog displays available agents
The job creation dialog SHALL fetch the agent directory from `/api/agents` and display each agent as an informational capability card showing name, description, and skill tags. Selecting a card SHALL NOT change the A2A routing target.

#### Scenario: Agents available
- **WHEN** user opens the New Job dialog and `/api/agents` returns one or more agents
- **THEN** the dialog displays agent cards with name, description, and skill tags as informational context
- **THEN** all A2A communication SHALL route to `studio-assistant` regardless of which card is highlighted

#### Scenario: No agents configured
- **WHEN** user opens the New Job dialog and `/api/agents` returns an empty array
- **THEN** the dialog displays the text input without capability cards
- **THEN** A2A communication SHALL route to `studio-assistant`

#### Scenario: Agent fetch fails
- **WHEN** user opens the New Job dialog and `/api/agents` returns an error
- **THEN** the dialog displays the text input without capability cards
- **THEN** A2A communication SHALL route to `studio-assistant`

## REMOVED Requirements

### Requirement: Selected agent is bound to the job
**Reason**: The assistant is now the permanent session owner. BYO agents are invoked by the assistant via delegation, not selected by the user.
**Migration**: Remove `agentName` from job store. All jobs route to `studio-assistant`. The assistant delegates to BYO agents via the `a2a_delegate` tool when needed.
