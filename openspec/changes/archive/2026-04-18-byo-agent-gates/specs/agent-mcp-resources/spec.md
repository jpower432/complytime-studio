## ADDED Requirements

### Requirement: Selective MCP resource enablement

Gemara MCP tool integration SHALL enable MCP resources on the Gemara
`McpToolset` only. ClickHouse MCP tool integration SHALL leave
`use_mcp_resources` at the default (`False`).

#### Scenario: Gemara McpToolset creation
- **WHEN** the gemara-mcp `McpToolset` is created
- **THEN** `use_mcp_resources` SHALL be `True`

#### Scenario: ClickHouse McpToolset creation
- **WHEN** the clickhouse-mcp `McpToolset` is created
- **THEN** `use_mcp_resources` SHALL be `False` (default)

### Requirement: Schema definitions via MCP resource protocol

The agent SHALL be able to obtain Gemara schema context by reading the
`gemara://schema/definitions` resource through the MCP resource protocol exposed
by the Gemara toolset.

#### Scenario: Agent needs Gemara schema context
- **WHEN** the agent requires Gemara schema context
- **THEN** it SHALL be able to read `gemara://schema/definitions` via MCP
  resources on the Gemara `McpToolset`
