## Why

kagent's Go runtime ignores `allowedHeaders` on `McpServer` tool references ([kagent#1679](https://github.com/kagent-dev/kagent/issues/1679)). This blocks On-Behalf-Of (OBO) token propagation from user sessions to the GitHub MCP server, forcing a static PAT workaround. The Python runtime already supports `allowedHeaders`, making it the viable path to per-user auth now.

## What Changes

- Switch all three Declarative Agent CRDs from `runtime: go` to `runtime: python` in the Helm template (`agent-specialists.yaml`).
- Restore `allowedHeaders: [Authorization]` on every `studio-github-mcp` tool reference.
- Retain the static `tokenSecret` on the GitHub MCPServer CRD as a fallback for local dev environments without user OAuth sessions.
- No changes to the canonical `agent.yaml` files (they already declared `allowedHeaders`; the Helm template was the only place it was stripped).

## Capabilities

### New Capabilities

- `python-runtime-obo`: Enables per-user GitHub token propagation through agent tool calls using kagent's Python runtime, which correctly handles `allowedHeaders`.

### Modified Capabilities

_(none -- no existing spec-level requirements change)_

## Impact

- **Helm chart**: `charts/complytime-studio/templates/agent-specialists.yaml` — runtime field and `allowedHeaders` restored.
- **Agent pods**: kagent will schedule Python-based agent containers instead of Go-based ones. Startup time and resource profile may differ.
- **Auth flow**: `Authorization: Bearer <token>` headers from the gateway OBO flow will now propagate to the GitHub MCP server through the Python runtime.
- **Upstream dependency**: Blocked on kagent#1679 for Go runtime parity. Once fixed upstream, switching back to Go runtime is a one-line change per agent.
