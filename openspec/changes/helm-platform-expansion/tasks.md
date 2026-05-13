# Tasks: Helm Chart — Platform Expansion

> Updated 2026-05 to reflect shipped state. Auth approach pivoted to
> OAuth2 Proxy sidecar (see `generic-oidc-auth`).

## New templates
- [x] Create `templates/postgres.yaml` (StatefulSet + Service + Secret, gated on `postgres.enabled`)
- [x] Add startup + readiness probes for PostgreSQL
- [ ] ~~Create `templates/langgraph-agents.yaml`~~ (deferred — blocked on LangGraph runtime design)
- [ ] ~~Create `templates/command-specs-configmap.yaml`~~ (deferred)
- [ ] ~~Create `templates/knowledge-base-mcp.yaml`~~ (deferred)
- [x] Create `templates/network-policies.yaml` (default-deny + per-component ingress)
- [x] Create `templates/cookie-secret.yaml` (auto-generated OAuth2 session secret)

## Modified templates
- [x] Update `templates/gateway.yaml` with OAuth2 Proxy sidecar + POSTGRES_URL env vars
- [x] Add initContainer for PostgreSQL readiness wait
- [ ] ~~Mount command-specs ConfigMap~~ (deferred — no ConfigMap created)
- [ ] ~~Update `templates/platform-prompts-configmap.yaml`~~ (deferred)

## Values
- [x] Add `auth.oauth2Proxy.*` section (replaces planned `auth.oidc.*`)
- [x] Add `postgres.*` section with `existingSecret` for production
- [ ] ~~Add `langgraphAgents.*` section~~ (deferred)
- [ ] ~~Add `rag.*` section~~ (deferred)
- [x] Expand `agentDirectory` with `id`, `a2a.skills` fields
- [ ] ~~Create preset files~~ (deferred)

## Docker Compose
- [x] Add PostgreSQL service to `docker-compose.yaml`
- [x] Add `POSTGRES_URL` to gateway environment
- [x] Add NATS and ORAS MCP services
- [x] Document Compose as local dev subset in README

## Validation
- [x] `helm template` renders correctly (default + ClickHouse enabled)
- [x] No stale references to old values paths
- [x] Secret references use `existingSecret` pattern consistently
