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
