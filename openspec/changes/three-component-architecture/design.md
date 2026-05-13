## Context

ComplyTime Studio is a Go modulith gateway that embeds a Preact SPA via `//go:embed`, serves it with history-mode fallback (`internal/web/serve.go`), and co-locates agent orchestration (A2A proxy, agent directory, MCP server configs). The workbench SPA communicates with the API via relative `/api/*` paths assuming same-origin deployment.

Two external consumer types are blocked by this coupling:
1. Internal dashboards that want to embed Studio UI components in their own tooling.
2. External agents (Cursor, Claude Code, custom) that need programmatic access to platform data.

The gateway currently serves: REST API, OAuth2 auth, A2A proxy, MCP proxy, SPA assets, and config. Agents access PostgreSQL directly via `postgres-mcp` (raw SQL).

## Goals / Non-Goals

**Goals:**

- Three independently deployable components in a monorepo: Platform (API), Studio (SPA), Agents
- Platform operates headlessly without UI assets
- Studio is a standalone static deployment with configurable API target
- Agents read platform data through domain-typed MCP resources, not raw SQL
- Dual API contracts: OpenAPI for human consumers, MCP resources for agent consumers
- Single Helm chart deploys all three with `studio.enabled` toggle
- Boundaries enforced by documentation and code review (POC)

**Non-Goals:**

- Multi-repo extraction (monorepo boundary enforcement only)
- Go SDK / typed client library for the REST API
- UI component library (`@complytime/studio-ui`) — requires signals-to-props refactor
- Automated cross-boundary import linting
- PostgreSQL store-layer refactors outside studio-mcp scope
- Agent framework changes (LangGraph stays)
- Auth model changes (OAuth2 Proxy stays)

## Decisions

### 1. Go code stays at repo root; boundary is conceptual

**Decision:** Keep `cmd/`, `internal/`, `go.mod` at the repo root. The "ComplyTime Platform" boundary is defined by documentation and what ships in the platform container image — not by a physical `complytime/` subdirectory.

**Rationale:** Moving Go code under a subdirectory rewrites every import path in the module (`github.com/complytime/complytime-studio/internal/...`). The churn provides no functional benefit in a monorepo where enforcement is via review, not tooling.

**Alternative rejected:** Physical `complytime/` directory. High churn, breaks all existing PRs, no enforcement mechanism beyond what docs provide.

### 2. Studio SPA uses runtime config injection (not build-time env vars)

**Decision:** At container startup, `envsubst` renders `env.js.template` into `env.js`, which sets `window.__STUDIO_CONFIG__ = { platformUrl: "..." }`. The SPA reads this at boot. `VITE_PLATFORM_URL` is also supported for local dev (`npm run dev`).

**Rationale:** A single container image works across all environments (dev, staging, prod) without rebuilding. Build-time env vars bake the API URL into the JS bundle.

**Alternative rejected:** Build-time `VITE_PLATFORM_URL` only. Requires separate builds per environment.

### 3. `studio-mcp` replaces `postgres-mcp` for agent data access

**Decision:** Introduce a `studio-mcp` server (Go binary under `cmd/studio-mcp/`) that imports `internal/store` and exposes platform data as typed MCP resources (`studio://policies`, `studio://evidence`, etc.). Agents no longer access PostgreSQL directly.

**Rationale:** Raw SQL access (via `postgres-mcp`) couples agents to the database schema. Domain-typed resources create a stable contract — the schema can change without breaking agents. Also reduces the agent's attack surface (no arbitrary SQL).

**Alternative rejected:** Keep `postgres-mcp` with a restricted query allowlist. Still couples agents to schema internals. Doesn't serve external agent consumers who shouldn't know the table structure.

### 4. Single Helm chart with three Deployment groups

**Decision:** One chart (`charts/complytime/`) deploys Platform, Studio, and Agent Deployments. `studio.enabled: true|false` toggles headless mode.

**Rationale:** POC needs to prove the boundary, not the operational independence. Separate charts add Helm dependency management and umbrella chart complexity. A single chart with feature toggles is simpler to develop and test.

**Alternative rejected:** Three charts + umbrella. Correct for production multi-team ownership but overkill for a POC validating boundaries.

### 5. Strangler extraction order: Studio first, then Agents

**Decision:** Phase 1 extracts Studio (remove embed, add Dockerfile, add CORS, write OpenAPI). Phase 2 introduces `studio-mcp` and updates agent config. Each phase produces a working system.

**Rationale:** Studio extraction is lower risk (no behavior change to agents or data layer). It validates the primary boundary (UI vs. API) before touching the agent-to-platform communication path.

**Alternative rejected:** Parallel extraction (both at once). Higher blast radius. If the MCP resource design is wrong, you've also broken the UI extraction branch.

## Risks / Trade-offs

| Risk | Mitigation |
|:--|:--|
| CORS misconfiguration blocks Studio in production | Default `CORS_ORIGINS` to Studio Service URL in Helm values. Integration test validates cross-origin `fetch` in CI. |
| `studio-mcp` resource surface is wrong (too granular or too coarse) | Start with 1:1 mapping to existing store interfaces. Iterate based on agent prompt needs. Resources are additive — add more later without breaking existing ones. |
| Agent prompt regression when switching from SQL to MCP resources | Run existing agent test suite against `studio-mcp` before removing `postgres-mcp`. Keep `postgres-mcp` as fallback behind a flag during transition. |
| Monorepo boundary erosion over time (no tooling enforcement) | Document rules in `AGENTS.md`. PR template checklist includes boundary check. Plan automated linting as fast-follow. |
| Runtime config injection (`env.js`) not picked up by SPA | Nginx serves `env.js` with `Cache-Control: no-cache`. SPA loads it synchronously before app boot. |
| Chart rename breaks existing `helm upgrade` for current users | This is a new branch / POC. Production migration guide written when merging to main. |

## Migration Plan

**Phase 1 — Extract Studio (on new branch off main):**

1. Create `studio/` directory structure from `workbench/`
2. Add `PLATFORM_URL` runtime config to SPA
3. Remove embed files and `internal/web/serve.go` from platform
4. Strip SPA wiring from `cmd/gateway/main.go`
5. Add `studio/Dockerfile` (Nginx) and `studio/nginx.conf`
6. Add CORS as mandatory config
7. Update Helm chart with Studio Deployment + `studio.enabled`
8. Update `docker-compose.yaml` with `studio` service
9. Write `docs/api/openapi.yaml`
10. Validate: Studio container calls Platform cross-origin, headless mode works

**Phase 2 — Extract Agents:**

1. Build `cmd/studio-mcp/` server
2. Add MCPServer CRD to Helm chart
3. Update `agents/assistant/agent.yaml` MCP block
4. Update agent prompt for `studio://` resources
5. Add `studio-mcp` to `docker-compose.yaml`
6. Write `docs/api/studio-mcp.md`
7. Rename chart to `charts/complytime/`
8. Validate: Agent reads data via `studio-mcp`, no `postgres-mcp` dependency

**Rollback:** Each phase is a set of commits on a feature branch. Rollback = revert the branch. No data migration involved.

## Open Questions

1. **Auth for `studio-mcp`:** Should `studio-mcp` authenticate requests (agent-to-MCP), or does the sidecar deployment model (same pod = trusted) make auth unnecessary? Leaning toward: no auth for stdio sidecar, token auth for HTTP standalone mode.
2. **Resource pagination:** Should `studio://evidence` support pagination parameters, or rely on `limit` only? Leaning toward: `limit` + `offset` query params on resources that can return large result sets.
3. **Chart naming timing:** Rename chart in Phase 2 or defer to merge-to-main? Leaning toward: Phase 2 (completes the POC narrative).
