## Context

The gateway implements GitHub OAuth and stores the user's access token in a session cookie. On A2A proxy requests, it extracts the token and injects `Authorization: Bearer <token>` into the outgoing request. Agent CRDs already declare `allowedHeaders: ["Authorization"]` on github-mcp tool references, so kagent propagates the header.

The problem: `github-mcp` runs as `stdio` transport. kagent starts it as a subprocess with `GITHUB_PERSONAL_ACCESS_TOKEN` baked into environment variables from a static Secret. The per-request `Authorization` header never reaches the MCP server — it's not wired to accept HTTP headers. Every user request hits GitHub with the same static token regardless of who's logged in.

The `github-mcp-server` binary supports two modes:
- `github-mcp-server stdio` — subprocess, reads `GITHUB_PERSONAL_ACCESS_TOKEN` from env
- `github-mcp-server http --port 3000` — HTTP server, accepts per-request `Authorization: Bearer` headers

## Goals / Non-Goals

**Goals:**
- Per-user GitHub token isolation for all github-mcp tool calls
- Zero static tokens in the deployment for GitHub access
- Private repo access scoped to the authenticated user's permissions

**Non-Goals:**
- Changing oras-mcp transport (already handled via gateway proxy)
- Modifying the OAuth flow (already requests `repo` scope at `internal/auth/auth.go:116`)
- Implementing fallback/anonymous GitHub access when unauthenticated

## Decisions

### D1: Switch github-mcp MCPServer CRD to streamablehttp transport

**Decision:** Change `spec.transportType` from `stdio` to `streamablehttp` and switch the command from `github-mcp-server stdio` to `github-mcp-server http --port 3000`.

**Rationale:** The `streamablehttp` transport creates an HTTP server that kagent calls via HTTP. kagent's `allowedHeaders` mechanism propagates the `Authorization` header from the A2A request to the MCP HTTP call. Each request gets the calling user's token.

**Alternative considered:** Custom sidecar proxy that intercepts stdio and injects tokens — rejected as unnecessary complexity given first-class HTTP support.

### D2: Remove static GITHUB_PERSONAL_ACCESS_TOKEN entirely

**Decision:** Delete the `lookup` for `studio-github-token` Secret, the `env` block injecting `GITHUB_PERSONAL_ACCESS_TOKEN`, and the `secretName`/`secretKey` values in `values.yaml`.

**Rationale:** A static token is a shared credential that grants the token owner's full access to every user. Removing it enforces that GitHub access is strictly per-user via OAuth. If the user is unauthenticated, the gateway blocks the request at HTTP 401 before it reaches the agent.

**Alternative considered:** Keep static token as fallback for unauthenticated/CI scenarios — rejected by user as a security risk.

### D3: Retain existing OAuth scopes

**Decision:** No changes to the OAuth flow. The gateway already requests `read:user,user:email,repo` scopes.

**Rationale:** The `repo` scope grants access to private repositories, which agents require. No scope expansion needed.

## Risks / Trade-offs

| Risk | Mitigation |
|:--|:--|
| Agents fail if user's OAuth token lacks `repo` scope (e.g., token was issued before scope was added) | Users re-authenticate to get a new token with `repo` scope. The gateway already requests it. |
| `github-mcp-server` HTTP mode behaves differently than stdio mode | The same codebase, same toolsets — only the transport changes. Toolset flags (`--toolsets`) work identically in both modes. |
| kagent `streamablehttp` transport not yet tested in this deployment | Validate with `helm template` and a local cluster before merging. The CRD schema is already documented in kagent docs. |

## Migration Plan

1. Update `mcp-servers.yaml` — switch transport and command args
2. Update `values.yaml` — remove `secretName`/`secretKey` under `mcpServers.github`
3. Update `setup.sh` — remove `studio-github-token` Secret creation
4. Update `README.md` — remove `GITHUB_TOKEN` from env vars, document that GitHub OAuth provides agent access
5. `helm template` to verify rendering
6. Deploy to kind cluster, verify agent tool calls use per-user tokens

**Rollback:** Revert the Helm chart changes and re-create the `studio-github-token` Secret.
