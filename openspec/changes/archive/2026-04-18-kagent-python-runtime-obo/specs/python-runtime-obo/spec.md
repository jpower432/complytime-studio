## ADDED Requirements

### Requirement: Agents use Python runtime
All Declarative Agent CRDs SHALL specify `runtime: python` in the `declarative` block.

#### Scenario: Agent pod uses Python runtime
- **WHEN** the Helm chart renders agent CRDs
- **THEN** every Agent resource MUST have `spec.declarative.runtime` set to `python`

### Requirement: OBO header propagation via allowedHeaders
Every Agent tool reference to an HTTP-transport MCP server requiring user identity SHALL include `allowedHeaders: [Authorization]`.

#### Scenario: GitHub MCP tool call includes user token
- **WHEN** a user initiates a job through the workbench
- **AND** the gateway injects an `Authorization: Bearer <token>` header on the A2A request
- **THEN** the agent's tool call to `studio-github-mcp` MUST forward that `Authorization` header

#### Scenario: No OBO token present (local dev)
- **WHEN** no `Authorization` header is present on the A2A request
- **AND** a static `tokenSecret` is configured on the MCPServer CRD
- **THEN** the agent MUST fall back to the static token for authentication

### Requirement: Static token fallback retained
The GitHub MCPServer CRD SHALL support an optional `tokenSecret` field that provides a static GitHub PAT via `secretRefs` for environments without OBO.

#### Scenario: tokenSecret configured
- **WHEN** `mcpServers.github.tokenSecret` is set in Helm values
- **THEN** the MCPServer CRD MUST include a `secretRefs` entry referencing that secret

#### Scenario: tokenSecret not configured
- **WHEN** `mcpServers.github.tokenSecret` is not set
- **THEN** the MCPServer CRD MUST NOT include `secretRefs`
