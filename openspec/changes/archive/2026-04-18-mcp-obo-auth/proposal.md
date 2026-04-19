## Why

github-mcp runs with a static `GITHUB_PERSONAL_ACCESS_TOKEN` shared across all agent invocations. Every user's request hits GitHub with the same token — wrong permission scope, shared rate limits, and a security risk where unauthenticated users inherit the token owner's access. The gateway already injects per-user OAuth tokens into A2A requests and `allowedHeaders` is already declared on agent CRDs, but the MCP server uses stdio transport which can't receive per-request headers.

## What Changes

- **Switch github-mcp from stdio to streamablehttp transport** — run `github-mcp-server http` instead of `github-mcp-server stdio`, accepting per-request `Authorization: Bearer` headers. **BREAKING**: removes `GITHUB_PERSONAL_ACCESS_TOKEN` env var and the `studio-github-token` Secret.
- **Remove static token configuration** — delete the Secret lookup, env injection, and `secretName`/`secretKey` values for github-mcp. No fallback static token.
- **No GitHub access without login** — agents calling github-mcp tools without an authenticated user session receive errors. gemara-mcp and clickhouse-mcp are unaffected (stdio, platform credentials).

## Capabilities

### New Capabilities

None — the OBO flow and transport specs already exist.

### Modified Capabilities

- `obo-token-forwarding`: Add requirement that no static token fallback exists; unauthenticated requests to github-mcp fail cleanly
- `mcpserver-crd-transport`: Implementation alignment — the spec already says streamablehttp, the codebase still says stdio

## Impact

- **Helm chart** (`mcp-servers.yaml`): github-mcp CRD switches transport, drops static token env
- **Helm chart** (`values.yaml`): Remove `mcpServers.github.secretName`/`secretKey` values
- **Deploy scripts** (`setup.sh`, `Makefile`): Remove `GITHUB_TOKEN` Secret creation for github-mcp
- **README**: Remove `GITHUB_TOKEN` from optional credentials, document that GitHub OAuth is required for github-mcp tools
- **No gateway changes** — token injection already works
- **No agent CRD changes** — `allowedHeaders` already declared
