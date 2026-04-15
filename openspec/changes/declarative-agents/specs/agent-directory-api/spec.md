## ADDED Requirements

### Requirement: Agent directory endpoint

The gateway SHALL expose a `GET /api/agents` endpoint that returns a JSON array of available specialist agent cards. Each card SHALL include the agent's name, description, A2A URL, and skills.

#### Scenario: List available agents

- **WHEN** a GET request is made to `/api/agents`
- **THEN** the response is a JSON array with one entry per deployed specialist, each containing `name`, `description`, `url`, and `skills` fields

#### Scenario: Agent card skills structure

- **WHEN** the agent directory response is inspected
- **THEN** each `skills` array entry contains `id`, `name`, `description`, and `tags` fields matching the A2A skill definition

### Requirement: Agent directory configured via values

The set of agents in the directory SHALL be configurable via `values.yaml`. The gateway SHALL read agent endpoints from environment variables or a configuration source.

#### Scenario: New specialist added to directory

- **WHEN** a new agent entry is added to `values.yaml` agent directory configuration
- **THEN** the gateway includes it in the `/api/agents` response after redeployment
