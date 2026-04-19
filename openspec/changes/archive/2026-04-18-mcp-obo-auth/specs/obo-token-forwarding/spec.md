## ADDED Requirements

### Requirement: No static token fallback for github-mcp
The github-mcp MCPServer CRD SHALL NOT include a static `GITHUB_PERSONAL_ACCESS_TOKEN` environment variable. GitHub access is exclusively via per-user OAuth tokens propagated through `allowedHeaders`.

#### Scenario: Unauthenticated agent request
- **WHEN** an A2A request reaches the agent without an `Authorization` header
- **THEN** the github-mcp tool call fails with a GitHub 401 error
- **THEN** the agent reports the tool failure to the user

#### Scenario: No static Secret in deployment
- **WHEN** the `studio-github-mcp` MCPServer CRD is rendered
- **THEN** the deployment spec contains no `env` block with `GITHUB_PERSONAL_ACCESS_TOKEN`
- **THEN** no `lookup` for `studio-github-token` Secret exists in the template

### Requirement: Setup scripts omit github-mcp static token
The `setup.sh` deployment script SHALL NOT create a `studio-github-token` Secret. The `GITHUB_TOKEN` environment variable SHALL NOT be referenced for MCP server configuration.

#### Scenario: Clean setup without GITHUB_TOKEN
- **WHEN** `setup.sh` runs without `GITHUB_TOKEN` set
- **THEN** no warning about "unauthenticated mode" is printed for github-mcp
- **THEN** no `studio-github-token` Secret is created

### Requirement: Values file omits github-mcp secret configuration
The `values.yaml` SHALL NOT contain `secretName` or `secretKey` fields under `mcpServers.github`.

#### Scenario: Default values
- **WHEN** `values.yaml` is rendered with defaults
- **THEN** `mcpServers.github` contains only `enabled` and `image` fields
