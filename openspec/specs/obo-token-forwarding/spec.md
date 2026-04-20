## ADDED Requirements

### Requirement: Gateway injects Authorization header on A2A requests
When proxying A2A requests to agent pods, the gateway SHALL extract the user's GitHub token from the session cookie and set the `Authorization: Bearer <token>` header on the outgoing request.

#### Scenario: Authenticated A2A request
- **WHEN** the gateway proxies an A2A request for an authenticated user
- **THEN** the outgoing request to the agent pod includes `Authorization: Bearer <user_github_token>`

#### Scenario: Unauthenticated A2A request
- **WHEN** an A2A proxy request has no valid session cookie
- **THEN** the gateway returns HTTP 401 without forwarding to the agent

### Requirement: Agent CRDs declare allowedHeaders for OBO MCP servers
Agent CRDs for agents using github-mcp or oras-mcp SHALL include `allowedHeaders: ["Authorization"]` on those tool references so kagent propagates the user's token from the A2A request to MCP tool calls.

#### Scenario: Threat modeler github-mcp OBO
- **WHEN** the studio-threat-modeler Agent CRD is rendered
- **THEN** the github-mcp tool reference includes `allowedHeaders: ["Authorization"]`
- **THEN** when the agent calls `get_file_contents`, kagent sends the user's Authorization header to github-mcp

### Requirement: github-mcp runs in HTTP mode with per-request tokens
The github-mcp MCPServer CRD SHALL use Streamable HTTP transport. The server accepts `Authorization: Bearer <token>` per request and creates an isolated server instance scoped to that token's permissions.

#### Scenario: Per-request token isolation
- **WHEN** two different users send requests through the same github-mcp server
- **THEN** each request is handled with its own GitHub token
- **THEN** user A sees only repos accessible to user A's token

### Requirement: oras-mcp runs in HTTP mode with per-request tokens
The oras-mcp MCPServer CRD SHALL use Streamable HTTP transport with per-request `Authorization` header support for OCI registry authentication.

#### Scenario: Per-request registry authentication
- **WHEN** an agent calls `list_repositories` with an Authorization header
- **THEN** oras-mcp authenticates to the OCI registry using the provided token
- **THEN** the response includes only repositories accessible to that user

### Requirement: Non-OBO MCP servers are unaffected
MCP servers without `allowedHeaders` (gemara-mcp, clickhouse-mcp) SHALL continue using stdio transport with platform-level credentials. No A2A request headers are forwarded to these servers.

#### Scenario: gemara-mcp unchanged
- **WHEN** the studio-gemara-mcp MCPServer CRD is rendered
- **THEN** it uses `transportType: stdio` with no `allowedHeaders`

### Requirement: Static token for MCP session initialization
The github-mcp MCPServer CRD SHALL include a `secretRefs` entry pointing to a Secret containing `GITHUB_PERSONAL_ACCESS_TOKEN`. This token authenticates MCP session initialization (tool discovery), which occurs before per-request `allowedHeaders` are available in the kagent Python runtime.

#### Scenario: tokenSecret configured
- **WHEN** `mcpServers.github.tokenSecret` is set in Helm values
- **THEN** the MCPServer CRD includes `secretRefs` referencing that Secret
- **THEN** MCP session initialization succeeds using the static PAT

#### Scenario: tokenSecret not configured
- **WHEN** `mcpServers.github.tokenSecret` is empty
- **THEN** no `secretRefs` appear on the MCPServer CRD
- **THEN** MCP session initialization fails with 401 and the agent cannot discover tools

### Requirement: OBO overrides static token for tool calls
When an authenticated user's `Authorization` header is propagated via `allowedHeaders`, tool calls SHALL use the user's OBO token rather than the static PAT. The static token is a baseline for session init only.

#### Scenario: Authenticated tool call
- **WHEN** an A2A request includes `Authorization: Bearer <user_token>`
- **AND** the agent calls `get_file_contents` on github-mcp
- **THEN** the tool call uses the user's token, not the static PAT

### Requirement: Values file supports github-mcp token configuration
The `values.yaml` SHALL contain a `tokenSecret` field under `mcpServers.github` defaulting to `"studio-github-token"`.

#### Scenario: Default values
- **WHEN** `values.yaml` is rendered with defaults
- **THEN** `mcpServers.github` contains `enabled`, `image`, and `tokenSecret` fields
