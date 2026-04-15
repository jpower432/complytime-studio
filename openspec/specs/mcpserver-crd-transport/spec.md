## MODIFIED Requirements

### Requirement: MCP server transport configuration
Each MCPServer CRD SHALL declare a transport type appropriate to its auth model. Servers requiring per-request user tokens SHALL use `streamablehttp` transport. Servers using platform credentials or no auth SHALL use `stdio` transport.

#### Scenario: github-mcp uses Streamable HTTP
- **WHEN** the studio-github-mcp MCPServer CRD is rendered
- **THEN** `spec.transportType` is `streamablehttp`
- **THEN** `spec.deployment.args` includes `http` subcommand with `--port` and `--toolsets` flags
- **THEN** the server listens on HTTP and accepts per-request `Authorization: Bearer` headers

#### Scenario: oras-mcp uses Streamable HTTP
- **WHEN** the studio-oras-mcp MCPServer CRD is rendered
- **THEN** `spec.transportType` is `streamablehttp`
- **THEN** the server accepts per-request `Authorization: Bearer` headers for OCI registry authentication

#### Scenario: gemara-mcp remains stdio
- **WHEN** the studio-gemara-mcp MCPServer CRD is rendered
- **THEN** `spec.transportType` is `stdio` (unchanged)

#### Scenario: clickhouse-mcp remains stdio
- **WHEN** the studio-clickhouse-mcp MCPServer CRD is rendered
- **THEN** `spec.transportType` is `stdio` (unchanged)
- **THEN** platform credentials are provided via Secret environment variables (unchanged)
