## ADDED Requirements

### Requirement: Manual A2A executor construction

The BYO gap analyst agent SHALL construct `A2aAgentExecutor` manually at startup
instead of calling `to_a2a()`. Configuration SHALL pass `A2aAgentExecutorConfig`
with a custom `adk_event_converter` as specified in the agent-artifact-emission
capability.

#### Scenario: Agent starts
- **WHEN** the agent starts
- **THEN** it SHALL construct `A2aAgentExecutor` manually (not via `to_a2a`)
- **THEN** it SHALL supply `A2aAgentExecutorConfig` including the custom
  `adk_event_converter`

### Requirement: Callback registration on LlmAgent

The gap analyst `LlmAgent` SHALL register all three deterministic gates:
`before_agent_callback`, `after_agent_callback`, and `before_tool_callback`.

#### Scenario: LlmAgent instantiation
- **WHEN** `LlmAgent` is created for the gap analyst
- **THEN** `before_agent_callback` SHALL be set
- **THEN** `after_agent_callback` SHALL be set
- **THEN** `before_tool_callback` SHALL be set

### Requirement: KMCP HTTP MCP connectivity

The agent SHALL connect to MCP services over streamable HTTP when URLs are
provided via environment variables. Missing URLs SHALL result in graceful
degradation (no hard failure of the process for optional integrations beyond
documented minimums).

#### Scenario: GEMARA_MCP_URL is set
- **WHEN** the `GEMARA_MCP_URL` environment variable is set
- **THEN** the agent SHALL connect using `StreamableHTTPConnectionParams` (or
  equivalent ADK HTTP MCP client configuration)

#### Scenario: CLICKHOUSE_MCP_URL is set
- **WHEN** the `CLICKHOUSE_MCP_URL` environment variable is set
- **THEN** the agent SHALL connect using `StreamableHTTPConnectionParams` (or
  equivalent ADK HTTP MCP client configuration)

#### Scenario: MCP URL unset
- **WHEN** either MCP URL environment variable is unset
- **THEN** the agent SHALL degrade gracefully (omit that toolset and continue
  with remaining configuration)
