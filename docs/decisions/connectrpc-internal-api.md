# SPDX-License-Identifier: Apache-2.0

# ADR 0026: ConnectRPC Internal API for studio-mcp

**Status:** Superseded — studio-mcp migrated to REST; ConnectRPC and internal port removed.
**Date:** 2026-05-13

## Context

studio-mcp and the gateway both import `internal/postgres/` and hold
independent database connections to the same PostgreSQL instance. This
creates two problems:

1. **No API boundary.** studio-mcp bypasses all gateway middleware —
   auth, write-protect, audit logging, rate limiting. A bug in
   studio-mcp can write directly to production tables with no gate.

2. **Tight binary coupling.** Both binaries must be built and deployed
   from the same Go module at the same version. A schema migration
   change can silently break studio-mcp at runtime if images drift
   during a rolling deploy.

studio-mcp is the **only** non-gateway component that needs direct
data access. The UI uses REST; the workbench uses MCP via studio-mcp.
The scope is one client, one server.

## Decision

Introduce a ConnectRPC internal API on the gateway's internal port
(:8081) and migrate studio-mcp from direct SQL to a generated
ConnectRPC client.

### Why ConnectRPC over alternatives

| Option | Contract enforcement | Build complexity | Debug ergonomics |
|:--|:--|:--|:--|
| Internal REST + OpenAPI | Medium (spec drift test) | Low | High (curl) |
| **ConnectRPC** | **High (proto codegen)** | **Medium** | **High (curl + JSON fallback)** |
| Full gRPC | High (proto codegen) | High | Low (grpcurl only) |

ConnectRPC uses `.proto` files for schema definition and code
generation (same as gRPC), but the transport is standard HTTP — you
can `curl` endpoints with JSON bodies. This gives compile-time
contract enforcement without requiring gRPC-specific load balancer
configuration or debugging tools.

Single Go dependency: `connectrpc.com/connect`.

### Architecture

```
Agent ──MCP──► studio-mcp ──ConnectRPC──► Gateway :8081 ──SQL──► PostgreSQL
```

studio-mcp no longer holds a `*sql.DB` connection. PostgreSQL has one
writer (the gateway). The gateway enforces validation, audit logging,
and write-protect on every path — public and internal.

### Proto definition

A single `.proto` file defines the internal contract:

```
proto/studio/v1/studio.proto
```

Covers the resources and tools studio-mcp currently exposes:

| Proto service method | Current studio-mcp path |
|:--|:--|
| `ListPolicies` | `studio://policies` |
| `GetPolicy` | `studio://policies/{id}` |
| `QueryEvidence` | `studio://evidence?policy_id=...` |
| `IngestEvidence` | `ingest_evidence` tool |
| `ListPosture` | `studio://posture?policy_id=...` |
| `ListCatalogs` | `studio://catalogs` |
| `ListMappings` | `studio://mappings?source_catalog=...` |
| `ListAuditLogs` | `studio://audit-logs` |
| `CreateDraftAuditLog` | `save_draft_audit_log` tool |
| `ListThreats` | `studio://threats` |
| `ListRisks` | `studio://risks` |

### Service layer refactor

Gateway handler logic is currently embedded in Echo handler functions.
To serve both REST and ConnectRPC, factor out a shared service layer:

```
Echo handler     ──┐
                   ├──► PolicyService.List() ──► internal/postgres/
Connect handler  ──┘
```

This is an incremental refactor — extract the core query/write logic
from each Echo handler into a service struct, then wire both Echo and
Connect handlers to the same service.

### Auth model for internal API

Network-enforced (Kubernetes NetworkPolicy restricts access to :8081).
No token or mTLS. Same model currently used for agent-to-gateway
traffic per `internal-endpoint-isolation.md` (ADR 0006).

### Where studio-mcp lives

Stays in `complytime-studio`. The `.proto` file, generated client, and
studio-mcp binary are all in the same Go module. A schema migration,
internal API handler update, and studio-mcp client update land in one
PR. Compile-time breakage if any side drifts.

## Consequences

**Positive:**
- `.proto` file **is** the data contract — field rename breaks both
  sides at compile time
- studio-mcp drops `internal/postgres/` import and `*sql.DB` — no
  more direct database access from a second binary
- Gateway enforces middleware on all write paths (internal and public)
- If studio-mcp ever moves to Python (`complytime-agents`), `protoc`
  generates a Python client from the same `.proto`
- Additive proto field evolution (field numbers) gives free forward
  compatibility

**Negative:**
- ~5-10ms added latency per MCP call (loopback HTTP) — negligible
  relative to LLM inference time
- New build dependency: `protoc` + `protoc-gen-go` +
  `protoc-gen-connect-go` (or `buf`)
- Service layer refactor touches all ~20 handler functions

**Neutral:**
- Internal API is not exposed publicly; Nginx only routes to :8080
- No impact on studio-ui, complytime-agents, or studio-deploy

## References

- ADR 0006: [Internal Endpoint Isolation](internal-endpoint-isolation.md)
- ADR 0025: [Data Platform + Workbench Split](data-platform-workbench-split.md)
- [ConnectRPC](https://connectrpc.com/)
