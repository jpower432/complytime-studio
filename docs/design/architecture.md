<!--
SPDX-License-Identifier: Apache-2.0
-->

# ComplyTime Studio Architecture

## Overview

ComplyTime Studio spans three deployable surfaces: the **gateway** (this repo, Go) is a **data platform API** only; the **Studio Workbench** ([complytime-studio](https://github.com/complytime-labs/complytime-studio), Python / Starlette) owns A2A, agent directory, chat, Gemara validate/migrate, and OCI publish/browse; the **Studio UI** ([studio-ui](https://github.com/complytime/studio-ui), Preact + Nginx) serves the browser SPA and reverse-proxies to gateway and workbench. The gateway reads and writes **PostgreSQL**, publishes and subscribes on **NATS**, and optionally uses **S3-compatible blob storage**. **ClickHouse** is an optional analytical tier reached from PostgreSQL via `pg_clickhouse` FDW when enabled—the gateway does **not** query ClickHouse directly.

---

## System architecture

```mermaid
flowchart TB
  subgraph Clients
    Browser["Browser"]
  end

  subgraph Nginx["studio-ui — Nginx"]
    Routes["/api/* /auth/* /oauth2/* → gateway · /workbench/* → workbench · /* → static SPA"]
  end

  subgraph Gateway["Gateway — public :8080 — Echo only"]
    Echo["Echo — middleware, auth, /api/*, /auth/*, /healthz"]
  end

  PG[("PostgreSQL — required")]
  NATS[("NATS — required")]
  Blob[("S3-compatible blob — optional")]
  CH[("ClickHouse — optional via FDW")]

  subgraph Workbench["complytime-studio — workbench"]
    WB["Starlette — A2A, agents, chat, validate/migrate, OCI"]
  end

  GemaraMCP["gemara-mcp"]
  StudioMCP["complytime-mcp"]
  OrasMCP["oras-mcp"]

  Assistant["Studio Assistant — LangGraph"]

  Browser --> Nginx
  Nginx --> Echo
  Nginx --> WB
  Echo --> PG
  Echo --> NATS
  Echo -->|"BLOB_*"| Blob
  WB --> GemaraMCP
  WB --> StudioMCP
  WB --> OrasMCP
  Assistant --> GemaraMCP
  Assistant --> StudioMCP
  Assistant --> OrasMCP
  PG -.->|"pg_clickhouse when configured"| CH
  NATS -->|"certification pipeline"| Echo
```

---

## Components

### Gateway (`cmd/gateway`)

**BLUF:** Echo serves **only** `/api/*`, `/auth/*`, and `/healthz` on `PORT` (default **8080**). Single port, single listener. No embedded SPA, no A2A proxy, no agent directory, no chat store, no validate/migrate proxy, no registry/publish HTTP API for OCI (those live in workbench).

| Concern | Implementation |
|:--|:--|
| HTTP | **Echo only**—no `http.ServeMux`, no mux catch-all |
| Data | `internal/store` + `internal/postgres`—single pool; `EnsureSchema` at startup |
| Events | `internal/events`—NATS; debounced certification pipeline on evidence subjects |
| Blobs | `internal/blob`—MinIO-compatible client when `BLOB_*` set |
| Catalog seed | Unified ingest: a seed job posts artifacts to `POST /api/ingest` (no startup registry seed in the gateway binary). OCI **bundle** unpack for imports uses `go-gemara/bundle` in store handlers—not MCP; `REGISTRY_INSECURE` and registry credentials apply when OCI import/pull is configured—not for seed |
| Auth | `internal/auth`—OAuth2 Proxy `X-Forwarded-*` headers; `auth.RequireWrite` with narrow bypasses (below) |

**Startup hard requirements:** missing **`POSTGRES_URL`** or **`NATS_URL`**, or failed connect/schema init, **exits the process**.

### Authentication

| Mode | Condition |
|:--|:--|
| **OAuth2 Proxy** | Sidecar handles OIDC, session cookies. Gateway reads `X-Forwarded-Email`, `X-Forwarded-User`, `X-Forwarded-Preferred-Username`, `X-Forwarded-Groups`. Optional `PROXY_SECRET` + `X-Proxy-Secret` strips spoofed headers from untrusted clients. |
| **Dev** | No proxy—no headers. `/api/*` (except `/api/config`) returns 401 without `X-Forwarded-Email` or `Authorization: Bearer <STUDIO_API_TOKEN>`. |

OAuth2 Proxy owns `/oauth2/start`, `/oauth2/callback`, `/oauth2/sign_out`. The gateway exposes `/auth/me`.

Non-GET `/api/*` passes through `writeProtect` → `auth.RequireWrite`, with **only** this bypass: **`POST /api/bootstrap`**.

### PostgreSQL

**Single application database** for policies, evidence, programs, jobs, users, audit logs, draft audit logs, certifications, mappings, catalogs, controls, guidance, threats, risks, posture aggregates, inventory, and related tables. No ClickHouse fallback for gateway data paths.

### NATS

| Subject pattern | Use |
|:--|:--|
| `core.evidence.<policy_id>` | After ingest—debounced **certification pipeline** |
| `core.draft.<policy_id>` | Published on draft creation (no active subscribers) |

### ClickHouse (optional)

**Not used by the gateway.** When operators enable ClickHouse and attach **PostgreSQL `pg_clickhouse` FDW**, analytics can target ClickHouse while the app stays Postgres-primary. See `docs/decisions/postgres-with-extensions.md`.

### Object storage (optional)

S3-compatible **MinIO API** for evidence attachments when `BLOB_ENDPOINT`, `BLOB_BUCKET`, and credentials are set.

### Studio Workbench and Assistant

**Workbench** ([complytime-studio](https://github.com/complytime-labs/complytime-studio)): Starlette app for **A2A routing**, **agent directory**, **chat state**, **`validate_gemara_artifact` / `migrate_gemara_artifact`** (via **gemara-mcp**), **OCI publish/browse** (via **oras-mcp**), and data access via **complytime-mcp** (`complytime://*` resources/tools). MCP servers are **not** invoked by the gateway.

**Studio Assistant** is a **LangGraph** agent in **complytime-studio** (not Google ADK). It uses the same MCP surface as other agents the workbench schedules.

### Studio UI

Preact SPA ([studio-ui](https://github.com/complytime/studio-ui)) in Nginx. Nginx sends **`/api/*`**, **`/auth/*`**, **`/oauth2/*`** to the gateway; **`/workbench/*`** to the workbench; **`/*`** to static assets. See `docs/decisions/studio-spa-extraction.md`.

---

## Data flow

```mermaid
sequenceDiagram
  participant C as Client
  participant G as Gateway
  participant PG as PostgreSQL
  participant N as NATS

  C->>G: POST /api/ingest
  G->>PG: insert evidence
  G->>N: publish core.evidence.policy_id
  N->>G: subscriber — certification pipeline
  G->>PG: certifications, evidence.certified flag
```

---

## Key routes (partial)

Registration: `internal/store/handlers.go`, `internal/auth/user_handlers.go`, `internal/auth/auth.go`, `cmd/gateway/main.go`.

| Method(s) | Path | Notes |
|:--|:--|:--|
| GET | `/healthz` | Public port; Postgres ping |
| GET | `/api/system-info` | Version, DB/auth hints |
| GET | `/api/config` | Non-secret config map |
| GET, POST | `/api/programs` | List, create |
| GET, PUT, DELETE | `/api/programs/{id}` | Read, update, delete |
| GET | `/api/programs/{id}/posture`, `/api/programs/posture` | Program posture |
| GET, POST | `/api/programs/{id}/jobs` | Jobs |
| GET, PATCH | `/api/jobs/{id}`, `/api/jobs/{id}/status` | Job read / status |
| GET | `/api/policies` | List |
| GET | `/api/policies/{id}` | Detail |
| POST | `/api/policies/import` | OCI policy import |
| POST | `/api/import` | Gemara bundle / artifact import |
| GET | `/api/catalogs` | List catalogs |
| GET | `/api/inventory` | Inventory |
| GET | `/api/evidence` | Query |
| POST | `/api/ingest` | Unified Gemara ingest (async job; NATS-backed worker + downstream publish) |
| GET | `/api/ingest/jobs/{job_id}` | Ingest job status |
| GET | `/api/audit-logs`, `/api/audit-logs/{id}` | Audit logs |
| POST | `/api/audit-logs` | Create |
| GET | `/api/draft-audit-logs`, `/api/draft-audit-logs/{id}` | Drafts |
| PATCH | `/api/draft-audit-logs/{id}` | Reviewer edits |
| POST | `/api/audit-logs/promote` | Promote draft |
| GET | `/api/requirements`, `/api/requirements/{id}/evidence` | Matrix + drill-down |
| GET | `/api/posture`, `/api/risks/severity`, … | Posture, risks, threats |
| POST | `/api/draft-audit-logs` | Create draft |
| GET | `/auth/me` | Identity from OAuth2 Proxy headers + user table |

---

## Configuration

### Environment (gateway)

| Variable | Required | Purpose |
|:--|:--|:--|
| `POSTGRES_URL` | **Yes** | Application database |
| `NATS_URL` | **Yes** | Event bus |
| `PORT` | No | 8080 default |
| `BLOB_*` | No | Object storage |
| `CORS_ORIGINS` | No | Comma-separated allowed origins |
| `PROXY_SECRET` | Prod recommended | Shared secret with OAuth2 Proxy for `X-Forwarded-*` trust |
| `REGISTRY_INSECURE` / registry credential env | No | OCI/registry pull when import is configured (not needed for unified seed via `/api/ingest`). See `internal/store/registry_config.go` |

`GEMARA_MCP_URL` and `ORAS_MCP_URL` are **workbench** concerns, not gateway env vars.

### Helm defaults (`studio-deploy` — `charts/complytime/values.yaml`)

Exact keys live in that repo. Typical toggles:

| Key | Default (typical) | Notes |
|:--|:--|:--|
| `postgres.enabled` | `true` | Application database |
| `nats.enabled` | `true` | Event bus |
| `clickhouse.enabled` | `false` | Optional FDW tier |
| Gateway / UI / workbench / assistant images | per chart | Separate images per component |

Resources deploy into **`{{ .Release.Namespace }}`**.

---

## Kubernetes (typical)

Chart source: **`studio-deploy`** (`charts/complytime/`).

| Kind | Name (pattern) | Notes |
|:--|:--|:--|
| Deployment | studio-gateway | **8080** |
| Service | studio-gateway | ClusterIP → 8080 |
| StatefulSet | studio-postgres | When `postgres.enabled` |
| Deployment | studio-nats | When `nats.enabled` |
| StatefulSet | studio-clickhouse | Only when `clickhouse.enabled` |
| Deployment | studio-ui | Nginx SPA; port **80** |
| Deployment | workbench | complytime-studio Starlette (separate from gateway) |
| Deployment | studio-assistant | LangGraph agent pod (chart names may vary) |

MCP servers (gemara, oras, complytime-mcp) run as workloads the **workbench** uses, not as gateway dependencies.

---

## Routing: public Echo

All HTTP routing is on a single **Echo** listener: middleware stack (recovery, request ID, security headers, optional CORS, Postgres degraded mode, auth, `writeProtect` → `RequireWrite`), then route groups.

---

## Removed: ClickHouse exports

PDF and Excel export functionality (`export_pdf.go`, `export_excel.go`, `export_common.go`) was removed during the ClickHouse-to-PostgreSQL migration. These were ClickHouse-specific and had no PostgreSQL equivalent. If export is needed, rebuild against the PostgreSQL schema.

---

## CI: PostgreSQL integration tests

Integration tests in `internal/store/` and `internal/postgres/` require a live PostgreSQL instance. Set `POSTGRES_TEST_URL` (e.g. `postgres://user:pass@localhost:5432/test?sslmode=disable`) to enable them. Without this variable, tests skip.

**Current state:** `.github/workflows/ci.yaml` does not provision a PostgreSQL service or set `POSTGRES_TEST_URL`, so integration tests are **always skipped in CI**. Run `make test-integration` locally with a Postgres instance to exercise SQL paths. CI pipelines **should** provision a PostgreSQL service to close this coverage gap.
