## Context

kagent supports two agent runtimes: Go and Python. Both implement the A2A protocol and MCP tool invocation, but the Go runtime has a known bug ([kagent#1679](https://github.com/kagent-dev/kagent/issues/1679)) where `allowedHeaders` in the `McpServer` tool configuration is silently ignored. This means `Authorization` headers from OBO flows never reach HTTP-transport MCP servers like `studio-github-mcp`.

The current workaround is a static GitHub PAT injected via `secretRefs` on the MCPServer CRD. This provides authentication but not per-user identity -- all agents act as a single GitHub user regardless of who initiated the request.

## Goals / Non-Goals

**Goals:**

- Restore per-user OBO token propagation from browser session through the gateway to the GitHub MCP server via agent tool calls.
- Minimize blast radius: one field change (`runtime: python`) per agent, no structural refactoring.
- Maintain backward compatibility for local dev via the existing static `tokenSecret` fallback.

**Non-Goals:**

- Migrating MCPServer CRDs from `v1alpha1` to a newer version.
- Implementing a gateway-side OBO token exchange (future consideration).
- Performance benchmarking Python vs Go runtime (accept kagent's Python runtime as-is).
- Fixing the Go runtime bug upstream (tracked in kagent#1679; revert to Go when fixed).

## Decisions

| # | Decision | Rationale | Alternatives Considered |
|:--|:--|:--|:--|
| 1 | Switch `runtime` from `go` to `python` | Python runtime is the only kagent runtime that propagates `allowedHeaders` to MCP HTTP calls. | (a) Wait for Go fix -- blocks OBO indefinitely. (b) Patch kagent locally -- high maintenance cost. |
| 2 | Restore `allowedHeaders: [Authorization]` on github-mcp tool refs | Required for the Python runtime to forward the `Authorization` header from the A2A request to MCP tool invocations. | (a) Keep static PAT only -- no per-user identity. |
| 3 | Retain `tokenSecret` on MCPServer CRD | Provides authentication for local/CI environments without user OAuth sessions (e.g., Kind clusters with no gateway auth). | (a) Remove it -- breaks local dev. |
| 4 | No changes to canonical `agent.yaml` | These files already declared `allowedHeaders`. The `runtime` field is a Helm/deploy concern, not a canonical agent property. | (a) Add `runtime` to `agent.yaml` -- introduces deployment coupling into the source-of-truth files. |

## Risks / Trade-offs

| Risk | Severity | Mitigation |
|:--|:--|:--|
| Python runtime has different performance characteristics (startup, memory) | Low | kagent manages pod lifecycle; agent pods are long-running. Startup latency is amortized. |
| Python runtime may have different streaming behavior | Medium | Tested with SSE streaming (`stream: true`); Python runtime produces `message/stream` events compatible with the workbench parser. |
| Static `tokenSecret` takes precedence over OBO header | Medium | Verify kagent's Python runtime merges headers (OBO takes priority when present). If not, document the precedence and conditionally disable `tokenSecret` in production. |
| Upstream Go fix lands and we forget to switch back | Low | Track kagent#1679 in the proposal. Revert is a one-line change (`runtime: go`) with `allowedHeaders` already in place. |
