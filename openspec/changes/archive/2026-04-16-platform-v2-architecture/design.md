## Context

ComplyTime Studio is a multi-agent platform for authoring Gemara GRC artifacts. Three specialist agents (threat modeler, gap analyst, policy composer) are deployed as kagent Declarative Agent CRDs. The gateway serves REST API proxies and an embedded workbench SPA. MCP servers provide tool access (gemara-mcp, github-mcp, oras-mcp, clickhouse-mcp).

Current state gaps: prompts contain duplicated platform constraints and embedded domain knowledge, agents use a shared GitHub PAT instead of per-user credentials, the frontend is vanilla JS without a component framework, and there is no user authentication.

## Goals / Non-Goals

**Goals:**

- Evolve the canonical `agent.yaml` spec to carry JTBD dimensions: skills (knowledge packs), allowedHeaders (OBO), composable prompt references
- Establish `platform.md` as the single source of shared agent identity and constraints
- Enable git-based skill packs from internal and external repositories
- Implement GitHub OAuth with per-user token forwarding to MCP servers via kagent's native `allowedHeaders`
- Replace the vanilla JS frontend with a React SPA
- Keep browser-side workspace state (Excalidraw model — no server-side persistence)
- Deploy everything via Helm to Kubernetes with kagent as the agent runtime

**Non-Goals:**

- Multi-tenancy or server-side workspace persistence
- Custom agent runtime (kagent Declarative CRDs only — no BYO binary)
- Non-Kubernetes local dev mode (kind/minikube is acceptable)
- Agent-to-agent orchestration (users chain specialists manually via UI)
- Creating external skill repositories (plan references now, create repos later)

## Decisions

### D1: JTBD agent spec with skills and allowedHeaders

Extend the canonical `agent.yaml` with two fields:

- `skills[]`: Git-based knowledge packs. Entries with only `path` are internal (this repo). Entries with `repo` + `ref` + `path` are external. Helm renders both to kagent `spec.skills.gitRefs[]`.
- `mcp[].allowedHeaders`: Header names to propagate from A2A requests to MCP tool calls. Helm renders to kagent `McpServerTool.allowedHeaders`.

**Alternative considered:** Embed all knowledge in prompts and use gateway-level MCP proxying for auth. Rejected — prompts become monolithic, gateway becomes a bottleneck for every MCP call, and the pattern doesn't leverage kagent's native capabilities.

### D2: Composable prompts via kagent promptTemplate

Use kagent's `promptTemplate.dataSources` to reference a `studio-platform-prompts` ConfigMap. Agent systemMessages use `{{include "platform/identity"}}` and `{{include "platform/constraints"}}` to compose shared fragments with agent-specific workflow instructions.

`agents/platform.md` is the source file, split into ConfigMap keys by the Helm template.

**Alternative considered:** Single monolithic ConfigMap key per agent (platform + agent concatenated at Helm render time). Rejected — loses kagent's native template variable support (`{{.AgentName}}`, `{{.ToolNames}}`, `{{.SkillNames}}`).

### D3: Internal vs. external skill split

Skills that encode ComplyTime Studio platform knowledge (gemara-layers, bundle-assembly, policy-risk-linkage) live in this repo under `skills/`. Skills that encode domain knowledge reusable by any compliance agent (stride-analysis, audit-classification, assessment-defaults) reference external repos.

Agent definitions reference external skills as `{ repo, ref, path }` in `agent.yaml`. Helm renders these to kagent `gitRefs` with the external URL. External repos are placeholders until created — agents degrade gracefully (fewer skills loaded, prompt still contains workflow).

**Alternative considered:** All skills internal. Rejected — domain methodology like STRIDE is not Studio-specific and should be shareable.

### D4: GitHub OAuth with JWT session cookie

Gateway implements GitHub OAuth App flow (`/auth/login`, `/auth/callback`, `/auth/me`). On successful auth, the user's GitHub access token is stored in an encrypted, signed JWT cookie (no server-side session store). The gateway remains stateless.

OAuth client ID and secret are provided via Kubernetes Secret, referenced in `values.yaml`.

**Alternative considered:** GitHub App (installation tokens) instead of OAuth App (user tokens). Rejected for now — OAuth App gives per-user tokens directly, which is what OBO needs. GitHub App tokens are scoped to the installation, not the user.

### D5: OBO via kagent allowedHeaders (Path 2)

github-mcp and oras-mcp switch from stdio to Streamable HTTP transport. Each MCP server runs in HTTP mode, accepting `Authorization: Bearer <token>` per request and creating an isolated server instance scoped to that token.

The gateway extracts the user's GitHub token from the JWT cookie and sets `Authorization: Bearer <token>` on outgoing A2A requests. kagent stores the header in session state. When the agent calls an MCP tool, kagent checks `allowedHeaders` on the tool reference and propagates matching headers from session state to the MCP request.

No custom proxy. No sidecar. Native kagent.

**Alternative considered:** Gateway-mediated MCP proxy (Path 1). The gateway would intercept MCP calls and inject tokens. Rejected — puts gateway in the hot path for every tool call, duplicates routing logic kagent already handles, and couples the gateway to MCP server topology.

### D6: React SPA with go:embed

Replace vanilla JS workbench with React SPA. Build output goes to `workbench/dist/`, embedded in the gateway binary via `go:embed`. Single container deployment.

UI surfaces: dashboard (agent cards from `/api/agents`), chat (A2A streaming per agent), artifact editor (Monaco/CodeMirror with live validation via `/api/validate`), publish panel (`POST /api/publish`), registry browser (`/api/registry/*`).

State lives in browser localStorage/IndexedDB. No server-side persistence.

**Alternative considered:** Separate frontend container (Nginx/Caddy). Rejected — adds a container and routing complexity for no benefit. `go:embed` keeps the deployment simple. Independent UI release cycles are not needed at this stage.

### D7: A2A reverse proxy in gateway

The gateway proxies A2A requests from the frontend to agent pods. Route: `POST /api/a2a/{agent-name}` → `http://{agent-name}:8080`. The gateway injects the `Authorization` header from the JWT cookie before forwarding.

The frontend never talks directly to agent A2A endpoints. This provides a single entry point, consistent auth injection, and the ability to add rate limiting or logging later.

**Alternative considered:** Frontend talks directly to agent A2A endpoints (exposed via Ingress). Rejected — leaks cluster topology to the browser, complicates auth injection, and requires per-agent Ingress rules.

### D8: MCP server transport matrix

| MCP Server | Transport | Auth Model | allowedHeaders |
|:-----------|:----------|:-----------|:---------------|
| gemara-mcp | stdio | None (no user data) | — |
| github-mcp | streamablehttp | OBO (user's GitHub token) | `[Authorization]` |
| oras-mcp | streamablehttp | OBO (user's registry token) | `[Authorization]` |
| clickhouse-mcp | stdio | Static (platform Secret) | — |

Only user-scoped MCP servers need HTTP transport. Platform-scoped servers stay on stdio.

## Risks / Trade-offs

| Risk | Mitigation |
|:-----|:-----------|
| kagent `allowedHeaders` is Python-runtime only today | Verify Go runtime support. If missing, use Python runtime for OBO agents or contribute the feature upstream. |
| github-mcp HTTP mode is new (merged Feb 2026) | Pin to a known-good release. Fall back to shared PAT if HTTP mode is unstable. |
| External skill repos don't exist yet | Agent definitions reference them as placeholders. Agents work without them — domain knowledge stays in prompts until skills are extracted. |
| React rewrite is significant frontend effort | Scope to MVP surfaces (dashboard, chat, editor). Publish panel and registry browser are stretch goals. |
| JWT cookie carries GitHub token (sensitive) | Encrypt cookie payload. Set HttpOnly, Secure, SameSite=Strict. Short expiry with refresh. |
| go:embed couples UI and gateway releases | Acceptable for current team size. Split later if release cadence diverges. |
| oras-mcp HTTP mode for OBO is unverified | Spike: confirm oras-mcp supports per-request auth headers. Fall back to gateway proxy (Path 1) for oras-mcp only if needed. |

## Open Questions

- **kagent Go runtime + allowedHeaders**: Does the Go ADK runtime propagate allowed headers from A2A session state to MCP calls? The Python runtime has `create_header_provider()` with `HEADERS_STATE_KEY`. Needs verification for Go.
- **OAuth token refresh**: GitHub OAuth tokens don't expire by default, but user can revoke. Strategy for handling revoked tokens mid-session?
- **oras-mcp HTTP mode**: Does oras-mcp-server support Streamable HTTP transport with per-request Bearer tokens? Needs spike.
- **Skill pack versioning**: When an external skill repo tags a release, should agent.yaml pin to a tag or track a branch?
