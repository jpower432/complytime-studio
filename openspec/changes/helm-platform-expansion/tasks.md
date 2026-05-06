# Tasks: Helm Chart — Platform Expansion

## New templates
- [ ] Create `templates/postgres.yaml` (StatefulSet + Service + Secret, gated on `postgres.enabled`)
- [ ] Add startup + readiness probes for PostgreSQL
- [ ] Create `templates/langgraph-agents.yaml` (single ranged template over `langgraphAgents`)
- [ ] Create `templates/command-specs-configmap.yaml` (glob `commands/*.md`)
- [ ] Create `templates/knowledge-base-mcp.yaml` (optional, gated on `rag.enabled`)
- [ ] Verify ConfigMap total size stays under etcd 1 MiB limit

## Modified templates
- [ ] Update `templates/gateway.yaml` with OIDC, POSTGRES_URL env vars
- [ ] Add initContainer for PostgreSQL readiness wait
- [ ] Mount command-specs ConfigMap at `/etc/studio/commands/`
- [ ] Update `templates/platform-prompts-configmap.yaml` with sub-agent directory block

## Values
- [ ] Add `auth.oidc.*` section, deprecate `auth.google.*`
- [ ] Add `postgres.*` section with `existingSecret` for production
- [ ] Add `langgraphAgents.*` section (separate from existing `agents.assistant`)
- [ ] Add `rag.*` section (optional, no default image)
- [ ] Expand `agentDirectory` with `id`, `role`, `framework`, `delegatable` fields
- [ ] Create preset files: `values-minimal.yaml`, `values-standard.yaml`, `values-full.yaml`

## Docker Compose
- [ ] Add PostgreSQL service to `docker-compose.yaml`
- [ ] Add `POSTGRES_URL` to gateway environment
- [ ] Document Compose as local dev subset in README

## Validation
- [ ] `helm template` renders correctly for minimal, standard, full profiles
- [ ] No stale references to old values paths
- [ ] Secret references use `existingSecret` pattern consistently
