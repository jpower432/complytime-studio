## Why

The gateway monolith mixes two fundamentally different traffic patterns in one binary: short-lived CRUD requests (data plane) and long-lived SSE streams to AI agents (agent plane). These scale differently, fail differently, and have zero shared state. Demonstrating the right architecture means cutting along the real boundary — not splitting the database layer horizontally.

`internal/agents/` imports only `consts` and `httputil`. No ClickHouse. No session state. It's already architecturally independent; currently colocated by accident.

## What Changes

- **Extract A2A proxy** into a standalone binary (`cmd/a2a-proxy/`) and Kubernetes Deployment
- **Strengthen gateway module boundaries** by splitting the monolithic `Store` into domain-specific interfaces (`EvidenceStore`, `PolicyStore`, `AuditLogStore`, `MappingStore`)
- **Document evidence extraction** as a future option enabled by the interface boundaries
- **Produce a decision record** capturing the rationale

## Capabilities

### New Capabilities

- `a2a-proxy-service`: Standalone stateless A2A reverse proxy with token propagation
- `service-decomposition`: Decision record documenting the chosen architecture

### Modified Capabilities

- `gateway-module-boundaries`: Store split into domain interfaces; no cross-imports between sibling packages
- `a2a-gateway-proxy`: A2A proxy extractable without changing the public HTTP contract

### Deferred Capabilities

- `evidence-service`: Documented as a future extraction option; not implemented in this change

## Impact

| Area | Change |
|:--|:--|
| `cmd/a2a-proxy/` | New binary — stateless A2A reverse proxy |
| `internal/agents/` | Shared between gateway (agent directory) and proxy (A2A relay) |
| `internal/store/` | Split into domain-specific interfaces |
| `charts/complytime-studio/` | New Deployment + Service for A2A proxy; ingress routes `/api/a2a/` to proxy |
| `Dockerfile.a2a-proxy` | New lightweight Dockerfile (no workbench assets) |
| `Makefile` | New build/image/deploy targets for proxy |
| `docs/decisions/` | Decision record for architecture choice |
