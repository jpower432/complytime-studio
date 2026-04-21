# Design: Backend architecture — modulith gateway + A2A proxy extraction

## Context

The gateway is a Go monolith (`cmd/gateway/main.go`) that wires HTTP domains into one process. Primary domain packages under `internal/` include `agents`, `auth`, `clickhouse`, `config`, `store`, and `proxy`, with supporting packages `httputil`, `consts`, `publish`, `registry`, `web`, and `ingest`. The binary serves the SPA, GitHub OAuth, Gemara MCP proxy, ORAS/registry and publish flows, store CRUD (policies, evidence, mappings, audit logs backed by ClickHouse), agent directory + A2A reverse proxy, and runtime config. All of this ships as a single binary behind one Kubernetes Deployment today.

## Goals

Demonstrate the **right architecture** for this system: clean separation between the data plane (gateway) and the agent plane (A2A proxy), with strong internal module boundaries in the gateway.

**Non-goals:** production-grade service mesh, distributed tracing at platform scale, rewriting components in another language, premature evidence service extraction.

## Key Insight: The Real Architectural Seam

The natural decomposition boundary is **not** between evidence writes and dashboard reads (same ClickHouse, same schema, same coupling). It's between the **data plane** and the **agent plane**:

- **Gateway (data plane):** SPA, OAuth, CRUD, evidence ingestion, registry, MCP proxy, config — everything that touches ClickHouse and user sessions. Cohesive. Stays as one binary.
- **A2A proxy (agent plane):** Stateless HTTP relay with token injection. No ClickHouse dependency. No session state. Independent failure domain. Independent scaling axis.

`internal/agents/` imports only `internal/consts` (timeout values) and `internal/httputil` (WriteJSON, TokenProvider). Zero database coupling. Already architecturally independent — currently colocated by accident, not by necessity.

## Decisions

### Decision 1: Extract A2A proxy as a standalone service

Move `internal/agents/` to its own binary (`cmd/a2a-proxy/`). The proxy is stateless, has no ClickHouse dependency, and scales on a different axis (long-lived SSE streams under concurrent chat load vs. short-lived CRUD requests).

| Pros | Cons |
|:-----|:-----|
| Scale chat load independently from CRUD | Two binaries, two Deployments |
| Agent unreachable ≠ dashboard down | Token propagation must be explicit (no shared cookie jar) |
| Trivially horizontal — no state, no DB | One more Helm template |
| Demonstrates correct service boundary | |

The only shared dependency is `TokenProvider` for OBO token injection. The proxy receives the token from the gateway (or ingress) and forwards it — no session store needed.

### Decision 2: Gateway stays a modulith with clean domain interfaces

The gateway keeps all data-plane concerns in one binary. Internally, split the monolithic `Store` struct into domain-specific types behind interfaces:

```
internal/store/evidence.go  → EvidenceStore interface
internal/store/policies.go  → PolicyStore interface
internal/store/auditlogs.go → AuditLogStore interface
internal/store/mappings.go  → MappingStore interface
```

This gives the **option** to extract evidence (or any domain) later by swapping the concrete implementation for an HTTP client — without paying the operational cost now.

### Decision 3: Evidence service extraction is a documented future option

Evidence ingestion (bursty writes, CSV uploads, future OTel) differs from read-heavy dashboard traffic. If measured load demonstrates contention, the evidence path can be extracted because the modulith interfaces make it a packaging change, not a rewrite. Until then, it stays in the gateway.

### Rejected: Extract evidence service now

Splitting evidence into its own binary adds two deploy targets sharing the same ClickHouse schema. Two processes contending on the same connection pool doesn't provide real isolation — it adds operational cost without measured benefit. The interface boundaries inside the modulith deliver the same code-level separation without the ops overhead.

### Rejected: Full decomposition

Separate binaries for gateway, A2A proxy, evidence, and store. Overkill for a prototype. Latency, auth propagation complexity, and debugging overhead aren't justified before team boundaries or load patterns demand it.

## Architecture

```
┌─────────┐     ┌───────────────────────────┐     ┌─────────────┐
│ Browser │────▶│   Gateway (modulith)      │────▶│ ClickHouse  │
│         │     │                           │     └─────────────┘
│         │     │ SPA + OAuth               │
│         │     │ /api/policies             │
│         │     │ /api/evidence             │
│         │     │ /api/audit-logs           │
│         │     │ /api/validate, /api/migrate│
│         │     │ /api/registry             │
│         │     │ /api/config               │
│         │     └───────────────────────────┘
│         │
│         │     ┌───────────────────────────┐     ┌─────────────┐
│         │────▶│   A2A Proxy (stateless)   │────▶│ BYO agents  │
│  SSE    │◀────│                           │     │ kagent      │
│ stream  │     │ /api/a2a/{agent}          │     └─────────────┘
│         │     │ token injection           │
└─────────┘     │ no DB, no session         │
                └───────────────────────────┘
```

## Risks

| Risk | Mitigation |
|:-----|:-----------|
| Token propagation breaks when proxy is separate | Explicit `Authorization` header contract; proxy accepts and forwards, no cookie parsing needed |
| Two Deployments increase ops surface | Proxy is trivially simple; health check is a TCP listen |
| Ingress routing adds complexity | Single path prefix `/api/a2a/` routes to proxy; everything else routes to gateway |
| Evidence extraction deferred too long | Interface boundaries make it a packaging change when needed |
