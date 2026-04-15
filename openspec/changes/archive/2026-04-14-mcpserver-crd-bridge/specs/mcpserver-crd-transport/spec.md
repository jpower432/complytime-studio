## ADDED Requirements

### Requirement: MCP servers deployed as MCPServer CRDs
The Helm chart SHALL deploy gemara-mcp, oras-mcp, and github-mcp as `MCPServer` custom resources (`kagent.dev/v1alpha1`) with `transportType: stdio` instead of plain Kubernetes Deployments and Services.

#### Scenario: Chart renders MCPServer resources
- **WHEN** `helm template` is run with default values
- **THEN** the output SHALL contain three `MCPServer` resources named `studio-gemara-mcp`, `studio-oras-mcp`, and `studio-github-mcp`
- **THEN** each resource SHALL have `spec.transportType: stdio`

#### Scenario: MCP server disabled via values
- **WHEN** `mcpServers.gemara.enabled` is set to `false`
- **THEN** the chart SHALL NOT render the `studio-gemara-mcp` MCPServer resource

### Requirement: KMCP controller bridges stdio to HTTP
The KMCP controller SHALL reconcile each MCPServer resource and produce a running Pod with an AgentGateway sidecar that bridges stdio to Streamable HTTP.

#### Scenario: MCPServer pod reaches Ready state
- **WHEN** a `MCPServer` resource with `transportType: stdio` is applied
- **THEN** the KMCP controller SHALL create a Deployment with the MCP container and an AgentGateway transport adapter
- **THEN** the resulting Pod SHALL reach `Ready` status within 120 seconds

#### Scenario: KMCP creates accessible Service
- **WHEN** the MCPServer pod is Ready
- **THEN** a Kubernetes Service SHALL exist that routes HTTP traffic to the AgentGateway sidecar port

### Requirement: Orchestrator connects to MCP servers via KMCP-generated Services
The orchestrator BYO agent env vars (`GEMARA_MCP_URL`, `ORAS_MCP_URL`, `GITHUB_MCP_URL`) SHALL point to the HTTP endpoints exposed by KMCP-generated Services.

#### Scenario: Orchestrator startup with MCP connectivity
- **WHEN** the orchestrator pod starts with `MCP_TRANSPORT=sse`
- **THEN** the orchestrator SHALL successfully connect to gemara-mcp and oras-mcp via their Service URLs
- **THEN** the orchestrator logs SHALL NOT contain "connection refused" or "proxy disabled" warnings for enabled MCP servers

#### Scenario: Orchestrator startup without GitHub token
- **WHEN** `GITHUB_TOKEN` is not set (placeholder secret)
- **THEN** the github-mcp pod SHALL still start
- **THEN** the orchestrator MAY log a degraded-mode warning for GitHub functionality

### Requirement: GitHub MCP server receives token via environment variable
The GitHub MCP server MCPServer resource SHALL inject `GITHUB_PERSONAL_ACCESS_TOKEN` from the `studio-github-token` Kubernetes Secret into the container environment.

#### Scenario: Token passed to github-mcp container
- **WHEN** the `studio-github-token` secret exists with key `token`
- **THEN** the github-mcp container SHALL have `GITHUB_PERSONAL_ACCESS_TOKEN` set from that secret value

### Requirement: No plain Deployment or Service resources for MCP servers
The chart SHALL NOT render `apps/v1 Deployment` or `v1 Service` resources for gemara-mcp, oras-mcp, or github-mcp. Those resources are managed by the KMCP controller.

#### Scenario: Chart output contains no MCP Deployments
- **WHEN** `helm template` is run with all MCP servers enabled
- **THEN** the output SHALL NOT contain Deployment resources with names `studio-gemara-mcp`, `studio-oras-mcp`, or `studio-github-mcp`
- **THEN** the output SHALL NOT contain Service resources with those names
