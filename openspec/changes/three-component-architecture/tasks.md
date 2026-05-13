## 1. Branch and Scaffold

- [x] 1.1 Create new branch `feat/three-component-arch` off `main`
- [x] 1.2 Create `studio/` directory structure (`src/`, `package.json`, `vite.config.ts`, `index.html`)
- [x] 1.3 Move `workbench/src/`, `workbench/package.json`, `workbench/vite.config.ts`, `workbench/index.html` to `studio/`
- [x] 1.4 Remove `workbench/` directory (including `embed.go`, `embed_dev.go`, `openspec/`)
- [x] 1.5 Move `workbench/openspec/` content to repo-root `openspec/` if not already there

## 2. Studio Runtime Config

- [x] 2.1 Create `studio/env.js.template` with `window.__STUDIO_CONFIG__ = { platformUrl: "${PLATFORM_URL}" }`
- [x] 2.2 Update `studio/src/api/fetch.ts` to read `window.__STUDIO_CONFIG__.platformUrl` and prepend to all fetch calls (default: empty string)
- [x] 2.3 Update `studio/src/api/a2a.ts` to use platformUrl for `/a2a/*` SSE connections
- [x] 2.4 Update `studio/src/api/auth.ts` to use platformUrl for auth redirects
- [x] 2.5 Add `VITE_PLATFORM_URL` support in `vite.config.ts` for local dev (`npm run dev`)

## 3. Studio Container

- [x] 3.1 Create `studio/nginx.conf` with history-mode fallback and no-cache for `env.js`
- [x] 3.2 Create `studio/Dockerfile` (multi-stage: Node build, Nginx serve)
- [x] 3.3 Add `studio/docker-entrypoint.sh` that runs `envsubst` on `env.js.template` then starts Nginx
- [x] 3.4 Verify: `docker build -t complytime-studio studio/` succeeds and produces a minimal image

## 4. Platform Headless Mode

- [x] 4.1 Delete `internal/web/serve.go`
- [x] 4.2 Remove `workbench` package import from `cmd/gateway/main.go`
- [x] 4.3 Remove `web.RegisterEchoWithMux()` call — replace with Echo default 404 handler returning JSON `{"error": "not found"}`
- [x] 4.4 Make `CORS_ORIGINS` configuration explicit in startup logs (warn if empty when not in dev mode)
- [x] 4.5 Verify: `go build ./cmd/gateway/` succeeds without `workbench/` or `internal/web/`
- [x] 4.6 Verify: `GET /` returns `404 {"error": "not found"}`, `GET /api/policies` returns 200

## 5. Docker Compose Integration

- [x] 5.1 Add `studio` service to `docker-compose.yaml` (build from `studio/`, port 3000, env `PLATFORM_URL=http://gateway:8080`)
- [x] 5.2 Verify: `docker compose up` starts gateway + studio + clickhouse + mcp servers
- [x] 5.3 Verify: Browser at `http://localhost:3000` loads SPA, env.js injects platform URL

## 6. Helm Chart — Studio Deployment

- [x] 6.1 Add `studio.enabled` and `studio.image` to `charts/complytime/values.yaml`
- [x] 6.2 Create `templates/studio-deployment.yaml` (Nginx container, `PLATFORM_URL` env var from platform Service URL)
- [x] 6.3 Create `templates/studio-service.yaml` (ClusterIP, port 80)
- [x] 6.4 Wrap Studio templates in `{{- if .Values.studio.enabled }}`
- [x] 6.5 Verify: `helm template` with `studio.enabled=true` renders Studio resources
- [x] 6.6 Verify: `helm template` with `studio.enabled=false` omits Studio resources

## 7. OpenAPI Spec

- [x] 7.1 Create `docs/api/openapi.yaml` with info block and server definitions
- [x] 7.2 Document policy endpoints (GET list, GET by ID, POST)
- [x] 7.3 Document evidence endpoints (POST ingest, GET query)
- [x] 7.4 Document audit-log endpoints (GET list, GET by ID, POST promote)
- [x] 7.5 Document draft-audit-log endpoints (GET list, PATCH)
- [x] 7.6 Document posture, mappings, requirements, catalogs, programs, agents, config, system-info
- [x] 7.7 Document notifications, certifications, threats, risks, validate, migrate
- [x] 7.8 Document auth endpoints (GET /api/me, POST /api/bootstrap)
- [x] 7.9 Define shared schemas (Policy, EvidenceRecord, AuditLog, PostureRow, error response)
- [x] 7.10 Define security schemes (bearerAuth, oauth2)

## 8. Studio-MCP Server

- [x] 8.1 Create `cmd/studio-mcp/main.go` scaffold (CLI with `--transport` and `--port` flags)
- [x] 8.2 Add ClickHouse connection setup (`internal/clickhouse`, `store.New`)
- [x] 8.3 Implement `studio://policies` resource (list and get-by-id)
- [x] 8.4 Implement `studio://evidence` resource with `policy_id`, `limit`, `offset` params
- [x] 8.5 Implement `studio://posture` resource with `policy_id` param
- [x] 8.6 Implement `studio://audit-logs` resource with `policy_id` param
- [x] 8.7 Implement `studio://mappings` resource with `source_catalog` param
- [x] 8.8 Implement `studio://catalogs` resource
- [x] 8.9 Implement `studio://threats` and `studio://risks` resources with `catalog_id` param
- [x] 8.10 Implement `ingest_evidence` tool
- [x] 8.11 Implement `save_draft_audit_log` tool
- [x] 8.12 Add stdio transport support
- [x] 8.13 Add HTTP transport support
- [x] 8.14 Create `Dockerfile.studio-mcp`
- [x] 8.15 Verify: `go build ./cmd/studio-mcp/` succeeds

## 9. Agent Layer Update

- [x] 9.1 Update `agents/assistant/agent.yaml` — replace `postgres-mcp` with `studio-mcp` in mcp block
- [x] 9.2 Update `agents/assistant/prompt.md` — replace SQL query references with `studio://` resource URIs
- [x] 9.3 Run `make sync-prompts` to copy updated prompt to chart
- [x] 9.4 Add `studio-mcp` MCPServer CRD template to Helm chart
- [x] 9.5 Add `studio-mcp` service to `docker-compose.yaml`
- [ ] 9.6 Verify: agent can read `studio://policies` and `studio://evidence` via studio-mcp

## 10. Helm Chart Rename and Finalize

- [x] 10.1 Rename `charts/complytime-studio/` to `charts/complytime/`
- [x] 10.2 Update `Chart.yaml` name to `complytime`
- [x] 10.3 Update Makefile references to chart path
- [x] 10.4 Verify: `helm template complytime ./charts/complytime/` renders all three Deployment groups

## 11. Documentation

- [x] 11.1 Create `docs/architecture.md` — three-component overview with communication diagram
- [x] 11.2 Create `docs/api/studio-mcp.md` — MCP resource URI reference
- [x] 11.3 Create `docs/decisions/three-component-architecture.md` ADR
- [x] 11.4 Create `docs/decisions/studio-spa-extraction.md` ADR
- [x] 11.5 Create `docs/decisions/studio-mcp-server.md` ADR
- [x] 11.6 Create `docs/decisions/agent-mcp-surface.md` ADR
- [x] 11.7 Update `AGENTS.md` with boundary rules section
- [x] 11.8 Update `README.md` with three-component architecture and local dev instructions
- [x] 11.9 Update Makefile with new targets: `studio-build`, `studio-image`, `studio-mcp-build`, `studio-mcp-image`

## 12. Validation

- [x] 12.1 `go build ./...` passes (no broken imports)
- [x] 12.2 `go test ./...` passes
- [x] 12.3 `cd studio && npm run build` succeeds (verified via Docker build)
- [x] 12.4 `docker compose up` starts all services and Studio can reach Platform
- [x] 12.5 `helm template` renders correctly with both `studio.enabled=true` and `studio.enabled=false`
- [ ] 12.6 Agent successfully reads data via `studio-mcp` (manual test)
