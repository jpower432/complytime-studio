<!--
SPDX-License-Identifier: Apache-2.0
-->

# ComplyTime Studio Architecture

## Overview

Studio is a compliance workbench: policies and catalogs from OCI registries, Gemara-native evidence ingestion, posture and requirement views, program-scoped jobs, and audit-log workflows backed by [Gemara](https://gemara.openssf.org/). The **gateway** (Go) is the only user-facing server; it reads and writes **PostgreSQL**, publishes and subscribes on **NATS**, and optionally uses **S3-compatible blob storage**. **ClickHouse** is an optional analytical tier reached from PostgreSQL via `pg_clickhouse` FDW when enabled—**the gateway does not talk to ClickHouse directly.**

---

## System architecture

```mermaid
flowchart TB
  subgraph Clients
    Browser["Browser — Workbench SPA"]
  end

  subgraph Gateway["Gateway — public :8080"]
    Echo["Echo — middleware, auth, CSP, CORS"]
    Mux["http.ServeMux — APIs, assets, registry, A2A"]
    Echo -->|echo.WrapHandler\(mux\) catch-all| Mux
  end

  subgraph Internal["Internal listener :8081"]
    IntMux["http.ServeMux — no auth"]
  end

  PG[("PostgreSQL — required")]
  NATS[("NATS — required")]
  Blob[("S3-compatible blob — optional")]
  OCI[("OCI registry")]
  CH[("ClickHouse — optional via FDW")]

  Assistant["Studio Assistant — ADK"]
  GemaraMCP["gemara-mcp"]
  PostgresMCP["postgres-mcp — pgEdge"]
  OrasMCP["oras-mcp"]

  Browser --> Echo
  Mux --> PG
  Mux --> NATS
  Mux -->|"GEMARA_MCP_URL"| GemaraMCP
  Mux -->|"ORAS_MCP_URL + insecure reg"| OCI
  Mux -->|"BLOB_*"| Blob
  Assistant --> PostgresMCP --> PG
  Assistant --> GemaraMCP
  Mux -->|"registry/publish"| OrasMCP --> OCI
  PG -.->|"pg_clickhouse when configured"| CH
  NATS -->|"posture + certification handlers"| PG
  NATS -->|"draft-audit-log → notifications"| PG
```

---

## Components

### Gateway (`cmd/gateway`)

**BLUF:** Echo on `PORT` (default **8080**) wraps application routes in `http.ServeMux`; a second **internal** HTTP server on `INTERNAL_PORT` (default **8081**) serves cluster-only paths with **no authentication**—protect it with `NetworkPolicy` (`NETWORKPOLICY_ENFORCED` documents that expectation).

| Layer | Role |
|:--|:--|
| Echo | Recovery, RequestID, security headers, optional CORS (`CORS_ORIGINS`), Postgres degraded middleware, auth middleware (reads OAuth2 Proxy `X-Forwarded-*` headers), `/api` user and chat-history groups, **`e.Any("/*", echo.WrapHandler(mux))`** for the embedded mux |
| Public mux | Store API, program API, posture (program-scoped), Gemara validate/migrate proxy, registry, publish, agent directory, A2A proxy/forward, config, embedded workbench assets |
| Internal mux | `POST /internal/draft-audit-logs`, lightweight `/healthz` |

**Hard requirements at startup:** missing **`POSTGRES_URL`** or **`NATS_URL`**, or failed connect/schema init, **exits the process**. NATS drives the certification pipeline and draft-audit-log notification path.

| Concern | Implementation |
|:--|:--|
| Data | `internal/store` + `internal/postgres` — single Postgres pool; `EnsureSchema` on startup |
| Events | `internal/events` — NATS publish/subscribe; debounced posture check + certification pipeline on evidence subjects |
| Blobs | `internal/blob` — MinIO-compatible client when `BLOB_*` set |
| Auth | `internal/auth` — see below |

### Authentication

| Mode | Condition |
|:--|:--|
| **OAuth2 Proxy** | `auth.oauth2Proxy.enabled: true` — sidecar handles OIDC discovery, PKCE, JWKS, token refresh, session cookies. Gateway reads identity from `X-Forwarded-Email`, `X-Forwarded-User`, `X-Forwarded-Preferred-Username`, `X-Forwarded-Groups` headers. |
| **Dev (no auth)** | `auth.oauth2Proxy.enabled: false` — sidecar omitted; no `X-Forwarded-*` headers; gateway middleware falls through to anonymous mode |

Any OIDC-compliant IdP works (Keycloak, Okta, Azure AD, Google, Dex, Hydra). Keycloak has a dedicated `--provider=keycloak-oidc` with role/group mapping.

OAuth2 Proxy owns `/oauth2/start`, `/oauth2/callback`, `/oauth2/sign_out`. The gateway exposes `/auth/me` (reads headers + user table). Non-GET `/api/*` is admin-gated via `writeProtect` middleware, with documented exceptions (chat history, A2A prefix, notification mark-read).

### PostgreSQL

**Single application database** for policies, evidence, programs, jobs, users, audit logs, draft audit logs, notifications, certifications, mappings, catalogs, controls, threats, risks, posture aggregates, and related tables. There is no ClickHouse fallback for gateway data paths.

### NATS

| Subject pattern | Use |
|:--|:--|
| `studio.evidence.<policy_id>` | After ingest — debounced **posture check** and **certification pipeline** |
| `studio.draft-audit-log.<policy_id>` | **Inbox notification** rows in Postgres when a draft audit log is created |

### ClickHouse (optional)

**Not used by the gateway.** When `clickhouse.enabled` is true in Helm, operators may attach **PostgreSQL `pg_clickhouse` FDW** so heavy analytical workloads can target ClickHouse while the app remains Postgres-primary. See `docs/decisions/postgres-with-extensions.md`. An optional **studio-clickhouse-mcp** CRD exists only when ClickHouse is enabled; the **assistant** ships with **postgres-mcp** and **gemara-mcp** only.

### Object storage (optional)

S3-compatible **MinIO API** for evidence attachments when `BLOB_ENDPOINT`, `BLOB_BUCKET`, and credentials are set.

### Studio Assistant

Python **Google ADK** agent: A2A server, MCP tools **`query_database`** / **`get_schema_info`** on **pgEdge postgres-mcp**, and **`validate_gemara_artifact`** / **`migrate_gemara_artifact`** on **gemara-mcp**. **oras-mcp** is a gateway-side MCP server for OCI registry operations (`ORAS_MCP_URL`), not an assistant tool in `agents/assistant/agent.yaml`.

### Workbench

Preact SPA embedded in the gateway (`go:embed`). Consumes the public REST API and A2A/SSE for the assistant.

---

## Data flow

```mermaid
sequenceDiagram
  participant C as Client
  participant G as Gateway mux
  participant PG as PostgreSQL
  participant N as NATS

  C->>G: POST /api/evidence/ingest
  G->>PG: insert evidence
  G->>N: publish studio.evidence.policy_id
  N->>G: subscriber — posture + certification
  G->>PG: posture updates, certifications, evidence flags

  Note over G,N: Draft audit log (internal or app path)
  G->>N: publish studio.draft-audit-log.policy_id
  N->>G: subscriber inserts notification
  G->>PG: notifications row
```

---

## Key routes (partial)

Exact registration lives in `internal/store/handlers.go`, `internal/postgres/handlers.go`, `internal/auth/auth.go`, `cmd/gateway/main.go`.

| Method(s) | Path | Notes |
|:--|:--|:--|
| GET | `/api/system-info` | Version, DB/auth/model hints |
| GET | `/api/config` | Published non-secret config map |
| GET, POST | `/api/programs` | List, create |
| GET, PUT, DELETE | `/api/programs/{id}` | Read, update, delete |
| GET | `/api/programs/{id}/posture`, `/api/programs/posture` | Program posture |
| GET, POST | `/api/programs/{id}/jobs` | Jobs under program |
| GET, PATCH | `/api/jobs/{id}`, `/api/jobs/{id}/status` | Job read / status |
| GET | `/api/policies` | List |
| GET | `/api/policies/{id}` | Detail |
| POST | `/api/policies/import` | OCI import |
| GET | `/api/evidence` | Query |
| POST | `/api/evidence/ingest` | Gemara-native ingest (+ NATS publish) |
| GET | `/api/audit-logs`, `/api/audit-logs/{id}` | Audit logs |
| POST | `/api/audit-logs` | Create |
| GET | `/api/draft-audit-logs`, `/api/draft-audit-logs/{id}` | Drafts |
| PATCH | `/api/draft-audit-logs/{id}` | Reviewer edits |
| POST | `/api/audit-logs/promote` | Promote draft |
| GET | `/api/requirements`, `/api/requirements/{id}/evidence` | Matrix + drill-down |
| GET | `/api/posture`, `/api/risks/severity`, … | Posture, risks, threats |
| GET | `/api/notifications` (+ unread-count, mark-read, create) | Inbox |
| POST | `/internal/draft-audit-logs` | **Internal port only** |
| GET | `/auth/me` | Identity from OAuth2 Proxy headers + user table |
|  | `/api/validate`, `/api/migrate` | Gemara MCP proxy when `GEMARA_MCP_URL` set |

---

## Configuration

### Environment (gateway)

| Variable | Required | Purpose |
|:--|:--|:--|
| `POSTGRES_URL` | **Yes** | Application database |
| `NATS_URL` | **Yes** | Event bus |
| `PORT` / `INTERNAL_PORT` | No | 8080 / 8081 defaults |
| `GEMARA_MCP_URL` | No | Validate/migrate proxy |
| `ORAS_MCP_URL` | No | Registry MCP |
| `BLOB_*` | No | Object storage |
| `CORS_ORIGINS` | No | Comma-separated allowed origins |
| `STUDIO_API_TOKEN` | No | Static bearer for scripts/CI |
| `NETWORKPOLICY_ENFORCED` | Prod | Acknowledges internal port locked down |

### Helm defaults (`charts/complytime-studio/values.yaml`)

| Key | Default | Notes |
|:--|:--|:--|
| `postgres.enabled` | `true` | Gateway image + **pgEdge postgres-mcp** image |
| `nats.enabled` | `true` | NATS for pipeline + notifications |
| `clickhouse.enabled` | `false` | Optional FDW / MCP tier |
| `gateway.image` | studio-gateway | |
| `assistant.image` | studio-assistant | |
| `mcpServers.gemara` / `oras` | enabled | |
| `registry.enabled` | `true` | Dev-oriented in-cluster registry |

Resources deploy into **`{{ .Release.Namespace }}`** (not a fixed namespace name).

---

## Kubernetes (typical)

| Kind | Name (pattern) | Notes |
|:--|:--|:--|
| Deployment | studio-gateway | Ports **8080** public, **8081** internal |
| Service | studio-gateway | ClusterIP → 8080 |
| Service | studio-gateway-internal | ClusterIP → 8081; NetworkPolicy-scoped |
| StatefulSet | studio-postgres | When `postgres.enabled` |
| Deployment | studio-nats | When `nats.enabled` |
| StatefulSet | studio-clickhouse | Only when `clickhouse.enabled` |
| MCPServer CRD | studio-gemara-mcp, studio-oras-mcp, studio-postgres-mcp | kagent; clickhouse MCP only if CH on |
| Deployment | studio-assistant | BYO agent |

Agents and extra MCP pods are scheduled by **kagent**; align `agentDirectory` and `agents/*/agent.yaml` with deployed servers.

---

## Routing: Echo + ServeMux hybrid

The gateway uses a **two-layer** routing model. Echo handles middleware (auth, CORS, recovery, degraded-mode) and owns `/auth/*`, `/api/*` routes registered via `internal/store/handlers.Register`. Everything else falls through to `http.ServeMux` via `echo.WrapHandler(mux)`:

| Layer | Owns | Why |
|:--|:--|:--|
| Echo (`e.Group("/api")`) | Evidence, policies, notifications, audit-logs, user management | Middleware stack (auth, write-protect, degraded) applied uniformly |
| ServeMux (`mux`) | Registry, publish, agents, config, A2A proxy, workbench assets, validate/migrate proxy | Existing handlers that predate Echo; wrapped 1:1 without rewrite |

**Rule:** new routes should use Echo unless they are pure proxies or static-file handlers. Both layers share the same `PORT`.

---

## Removed: ClickHouse exports

PDF and Excel export functionality (`export_pdf.go`, `export_excel.go`, `export_common.go`) was removed during the ClickHouse-to-PostgreSQL migration. These were ClickHouse-specific and had no PostgreSQL equivalent. If export is needed, rebuild against the PostgreSQL schema.

---

## CI: PostgreSQL integration tests

Integration tests in `internal/store/` and `internal/postgres/` require a live PostgreSQL instance. Set `POSTGRES_TEST_URL` (e.g. `postgres://user:pass@localhost:5432/test?sslmode=disable`) to enable them. Without this variable, tests skip. CI pipelines **should** provision a PostgreSQL service and set this variable to avoid permanent coverage gaps in evidence, policy, and user SQL paths.
