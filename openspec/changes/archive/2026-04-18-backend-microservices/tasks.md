## 1. Module boundary audit

- [x] 1.1 Map import graph for each `internal/` package — list direct dependencies on other internal packages
- [x] 1.2 Identify shared types used across packages (structs, interfaces, constants)
- [x] 1.3 Confirm `internal/agents/` has zero ClickHouse or session coupling
- [x] 1.4 Document coupling matrix in a table: package × package with dependency type

## 2. Gateway modulith — store interface split

- [x] 2.1 Define `EvidenceStore` interface in `internal/store/`
- [x] 2.2 Define `PolicyStore` interface in `internal/store/`
- [x] 2.3 Define `AuditLogStore` interface in `internal/store/`
- [x] 2.4 Define `MappingStore` interface in `internal/store/`
- [x] 2.5 Update `handlers.go` to accept interfaces instead of concrete `*Store`
- [x] 2.6 Update `cmd/gateway/main.go` wiring to pass concrete impls satisfying interfaces

## 3. A2A proxy extraction

- [x] 3.1 Create `cmd/a2a-proxy/main.go` — standalone binary registering `/api/a2a/{agent}` routes
- [x] 3.2 Extract token propagation: proxy accepts `Authorization` header and forwards to agents
- [x] 3.3 Add agent directory config via `AGENT_DIRECTORY` env var (same JSON as gateway)
- [x] 3.4 Add `KAGENT_A2A_URL` / `KAGENT_AGENT_NAMESPACE` config support
- [x] 3.5 Create `Dockerfile.a2a-proxy` (lightweight, no workbench assets)
- [x] 3.6 Add Helm template: Deployment, Service for `studio-a2a-proxy`
- [x] 3.7 Update ingress/routing: `/api/a2a/` routes to proxy Service
- [x] 3.8 Remove `/api/a2a/` route registration from gateway; keep `GET /api/agents` directory
- [x] 3.9 Add `a2a-proxy-image` and `a2a-proxy-build` targets to Makefile
- [x] 3.10 Update `deploy` target to build, load, and restart the proxy

## 4. Workbench integration

- [x] 4.1 Update `a2a.ts` endpoint to route through proxy (verify path unchanged, only backend routing differs)
- [ ] 4.2 Verify SSE streaming works through the proxy Deployment

## 5. Decision record

- [x] 5.1 Write decision record at `docs/decisions/backend-architecture.md`
- [x] 5.2 Document future evidence extraction path enabled by interface boundaries
