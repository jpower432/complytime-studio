# Backend Architecture: Modulith Gateway + Extracted A2A Proxy

**Status:** Accepted
**Date:** 2026-04-18

## Context

The gateway is a Go monolith serving six concerns: SPA, OAuth, A2A proxy, MCP proxy, store CRUD, and config. As the audit dashboard grows, we evaluated whether decomposition improves development velocity, scalability, or fault isolation.

## Decision

**Modulith gateway + extracted A2A proxy.**

- The A2A reverse proxy moves to a standalone binary (`cmd/a2a-proxy/`). It is stateless, has no ClickHouse dependency, and scales on a different axis (long-lived SSE streams vs. short-lived CRUD).
- The gateway stays as a single binary with domain-specific store interfaces (`EvidenceStore`, `PolicyStore`, `AuditLogStore`, `MappingStore`) that enable future extraction without code changes.

## Architecture

```
Browser ──▶ Gateway (modulith)  ──▶ ClickHouse
            │ SPA + OAuth
            │ /api/policies
            │ /api/evidence
            │ /api/audit-logs
            │ /api/validate
            │ /api/registry
            │
            ├── /api/a2a/ ──forward──▶ A2A Proxy (stateless) ──▶ Agents
```

## Rationale

The natural architectural seam is between the **data plane** (gateway) and the **agent plane** (A2A proxy), not within the data layer.

`internal/agents/` imports only `internal/consts` and `internal/httputil`. Zero database coupling, zero session state. Already architecturally independent — colocated by accident, not necessity.

Splitting evidence into its own binary adds two deploy targets sharing the same ClickHouse schema. Two processes contending on the same connection pool provides no real isolation.

## Rejected Alternatives

| Option | Reason |
|:--|:--|
| Extract evidence service | Shared ClickHouse schema negates isolation benefit. Interface boundaries in the modulith deliver the same code-level separation without ops overhead. |
| Full decomposition (gateway, A2A, evidence, store) | Overkill for prototype. Auth propagation, debugging complexity, and latency overhead not justified before team or load boundaries demand it. |
| Keep monolith with no changes | Misses the opportunity to demonstrate correct service boundaries between data and agent planes. |

## Future Options

Evidence extraction is enabled by the domain store interfaces. When measured load demonstrates contention between ingestion and dashboard reads, swap the ClickHouse-backed `EvidenceStore` implementation for an HTTP client pointing at a standalone evidence service. This is a packaging change, not a rewrite.

## Consequences

- Two Kubernetes Deployments instead of one (gateway + proxy)
- Gateway forwards `/api/a2a/` to the proxy service, keeping the frontend unchanged
- Proxy scales independently under chat load
- Agent unavailability does not affect dashboard CRUD
