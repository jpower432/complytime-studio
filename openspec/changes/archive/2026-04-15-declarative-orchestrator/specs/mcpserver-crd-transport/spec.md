## MODIFIED Requirements

### Requirement: Orchestrator connects to MCP servers via KMCP-generated Services
The declarative orchestrator Agent CRD SHALL reference MCP servers via `type: McpServer` tool entries instead of environment variable URLs. The KMCP-generated Services remain unchanged.

#### Scenario: Orchestrator startup with MCP connectivity
- **WHEN** the declarative orchestrator pod starts
- **THEN** kagent SHALL resolve `McpServer` tool references to the KMCP-generated Service endpoints
- **THEN** the orchestrator SHALL successfully connect to oras-mcp tools
- **THEN** the orchestrator logs SHALL NOT contain MCP connectivity errors for enabled MCP servers

#### Scenario: Orchestrator startup without GitHub token
- **WHEN** `GITHUB_TOKEN` is not set (placeholder secret)
- **THEN** the github-mcp pod SHALL still start
- **THEN** the orchestrator MAY log a degraded-mode warning for GitHub functionality
