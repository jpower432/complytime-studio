## Why

ComplyTime Studio conflates three concerns in a single binary: data platform (API, PostgreSQL, events, certifier), agent orchestration (A2A, MCP, kagent), and workbench UI (Preact SPA). This coupling prevents external consumers — internal dashboards that want to embed UI components, and external agents that need programmatic data access — from using the platform independently. The architecture needs clean boundaries so each layer can evolve, deploy, and be consumed independently.

## What Changes

- **BREAKING**: Gateway no longer serves the SPA. Non-API paths return 404 instead of `index.html`.
- Extract the workbench SPA into `studio/` as a standalone deployable (Nginx container).
- SPA calls the platform via configurable `PLATFORM_URL` (cross-origin, CORS required).
- Introduce `studio-mcp` server exposing platform data as MCP resources for agent consumers.
- Agents consume platform data through `studio-mcp` instead of `postgres-mcp` (no more raw SQL).
- Helm chart deploys three independent Deployment groups with `studio.enabled` toggle for headless mode.
- Dual API contracts: OpenAPI spec (REST) for human/CI consumers, MCP resource definitions for agents.
- Rename `charts/complytime-studio/` to `charts/complytime/`.

## Capabilities

### New Capabilities

- `studio-spa-deployment`: Standalone SPA deployment with runtime config injection, Nginx serving, and PLATFORM_URL-based API communication.
- `studio-mcp-server`: MCP server exposing platform data (policies, evidence, posture, audit logs, mappings, catalogs, threats, risks) as typed resources and tools for agent consumption.
- `headless-platform`: Platform gateway operating without UI assets — pure API server with `studio.enabled` toggle in Helm.
- `platform-openapi-contract`: OpenAPI 3.1 specification documenting the full REST API surface.

### Modified Capabilities

- `a2a-gateway-proxy`: Agent A2A routing unchanged but agents now use `studio-mcp` for data access instead of `postgres-mcp`.
- `gateway-module-boundaries`: Physical boundary enforcement — Studio removed from Go build, agent layer communicates only via MCP/REST.

## Impact

- **Go binary**: Removes `workbench/embed.go`, `internal/web/serve.go`, `workbench` package import from `cmd/gateway/main.go`.
- **Frontend**: All `src/api/*.ts` files updated to use configurable base URL.
- **Agents**: `agents/assistant/agent.yaml` MCP block changes from `postgres-mcp` to `studio-mcp`. Prompt updated for `studio://` resource URIs.
- **Helm chart**: Restructured with three Deployment groups. Chart renamed. New `studio-mcp` MCPServer CRD.
- **docker-compose.yaml**: New `studio` and `studio-mcp` services.
- **CI/CD**: New build targets for Studio image and studio-mcp image.
- **CORS**: Becomes mandatory configuration (platform must allow Studio origin).
