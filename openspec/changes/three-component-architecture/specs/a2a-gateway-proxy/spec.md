## MODIFIED Requirements

### Requirement: Agents access platform data via studio-mcp
Agents SHALL read platform data (policies, evidence, posture, audit logs, mappings, catalogs, threats, risks) through the `studio-mcp` MCP server. Agents SHALL NOT use `postgres-mcp` for direct database access.

#### Scenario: Agent queries evidence for gap analysis
- **WHEN** the assistant agent needs evidence data for a policy
- **THEN** it reads `studio://evidence?policy_id=<id>` via `studio-mcp`
- **THEN** it does NOT execute raw SQL via `postgres-mcp`

#### Scenario: Agent queries posture
- **WHEN** the assistant agent needs posture aggregates
- **THEN** it reads `studio://posture?policy_id=<id>` via `studio-mcp`

#### Scenario: postgres-mcp is removed from agent MCP surface
- **WHEN** the agent's `agent.yaml` is rendered into the kagent CRD
- **THEN** no `postgres-mcp` server reference is present
- **THEN** `studio-mcp` is listed with resource and tool access
