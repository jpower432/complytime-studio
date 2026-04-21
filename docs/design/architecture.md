# ComplyTime Studio Architecture

## Overview

Audit dashboard for compliance posture tracking, evidence synthesis, and agentic gap analysis. Stores policies imported from OCI registries, ingests evidence via API/file upload (OTel collector integration planned), and uses a BYO ADK assistant agent to produce Gemara AuditLog artifacts.

Artifact authoring (ThreatCatalogs, ControlCatalogs, Policies) is handled by engineers using local tooling + gemara-mcp. Studio focuses on the **consumption and analysis** side of the compliance lifecycle.

## System Diagram

```
Browser
└── Workbench (Preact SPA, hash-routed)
    ├── WorkspaceView   — CodeMirror YAML editor, validate, publish
    ├── ChatDrawer      — SSE stream to assistant, artifact extraction
    └── ImportDialog    — OCI registry browser, import refs

        │ HTTP (same-origin, studio_session cookie)
        ▼
Gateway (Go, :8080)
├── /                     → Embedded SPA (go:embed)
├── /api/policies         → Policy CRUD (ClickHouse)
├── /api/evidence         → Evidence query + ingestion
├── /api/audit-logs       → AuditLog history
├── /api/mappings         → Cross-framework crosswalks
├── /api/agents           → Agent directory (static JSON)
├── /api/a2a/{agent}      → Embedded A2A proxy → agent pod
├── /api/validate         → gemara-mcp proxy (Streamable HTTP)
├── /api/migrate          → gemara-mcp proxy (Streamable HTTP)
├── /api/registry/*       → oras-mcp proxy (or direct OCI API)
├── /api/publish          → OCI bundle assembly + push
├── /api/config           → Platform configuration
├── /auth/*               → Google OAuth flow (optional)
└── /healthz              → ClickHouse ping

Agent Pod (Python ADK, :8080)
└── studio-assistant      → Evidence-backed AuditLog from ClickHouse
    └── MCP: gemara-mcp, clickhouse-mcp

MCP Servers (kagent MCPServer CRDs via KMCP)
├── studio-gemara-mcp     → Gemara schema validation + migration
├── studio-oras-mcp       → OCI registry operations
└── studio-clickhouse-mcp → ClickHouse evidence queries

Infrastructure
├── ClickHouse            → Evidence, policies, mappings, audit logs
└── OCI Registry          → Gemara artifact bundles (Zot for dev)
```

## Components

### Gateway (Go)

User-facing entry point. Serves the embedded Preact SPA, REST APIs, Google OAuth, and proxies for MCP tools and OCI registries. The A2A proxy is embedded in the gateway when `A2A_PROXY_URL` is unset.

| Concern | Implementation |
|:--|:--|
| HTTP server | `net/http.ServeMux` with middleware chain (auth, CORS, security headers) |
| Data access | `internal/store` interfaces (`PolicyStore`, `EvidenceStore`, `AuditLogStore`, `MappingStore`) backed by ClickHouse |
| Authentication | Google OAuth (OpenID Connect) with AES-GCM encrypted session cookies |
| A2A proxy | Embedded reverse proxy to agent pods; standalone extraction via `A2A_PROXY_URL` |
| MCP proxy | Streamable HTTP client to gemara-mcp |
| OCI operations | ORAS MCP for secure registries, direct HTTP for insecure (dev) registries |
| Schema init | `EnsureSchema` creates tables on startup (90 retries, 2s backoff) |

### Studio Assistant (Python)

BYO ADK agent built with Google Agent Development Kit. Runs as a standalone Kubernetes Deployment.

| Concern | Implementation |
|:--|:--|
| Framework | Google ADK `LlmAgent` + `A2aAgentExecutor` + Starlette/Uvicorn |
| Model | Configurable via `MODEL_NAME` env var |
| Tools | MCP toolsets for gemara-mcp (with resources) and clickhouse-mcp |
| Callbacks | `before_agent` (input validation), `after_agent` (artifact extraction), `before_tool` (SQL injection guard) |
| Skills | Loaded from `/app/skills/*/SKILL.md` at startup, appended to system prompt |

### ClickHouse

Primary datastore. Deployed as a StatefulSet with PVC.

| Table | Engine | Partition | TTL | Purpose |
|:--|:--|:--|:--|:--|
| `evidence` | ReplacingMergeTree | `toYYYYMM(collected_at)` | configurable | Evaluation and enforcement results |
| `policies` | ReplacingMergeTree | — | — | Imported policy artifacts |
| `mapping_documents` | ReplacingMergeTree | — | — | Cross-framework crosswalks |
| `audit_logs` | ReplacingMergeTree | `toYYYYMM(audit_start)` | configurable | AuditLog artifacts from assistant |

Schema is created by the gateway's `EnsureSchema` on startup. The `evidence` table is aligned to the `beacon.evidence` OTel semantic convention — see [evidence-semconv-alignment.md](evidence-semconv-alignment.md).

### MCP Servers

Deployed via kagent MCPServer CRDs. The assistant and gateway connect over Streamable HTTP.

| Server | Purpose |
|:--|:--|
| gemara-mcp | Schema validation, artifact migration, Gemara resource access |
| clickhouse-mcp | SQL queries against evidence/audit data |
| oras-mcp | OCI registry operations |

### Workbench (Preact SPA)

Embedded in the gateway binary at build time. Editor-first workspace.

| Feature | Description |
|:--|:--|
| Workspace Editor | CodeMirror YAML editor with validate, download, copy, publish toolbar |
| Chat Drawer | Slide-out panel for agent conversations; artifacts stream into editor |
| Registry Import | Browse OCI registries, inspect layers, import mapping references |
| Definition Picker | User-selectable Gemara definition for validation |
| Theme Toggle | Dark/light mode with system preference detection |

## Evidence Pipeline

Two intake paths feed a single `evidence` table.

| Path | Source | How | Enrichment |
|:-----|:-------|:----|:-----------|
| A — Gemara-native | `complyctl` via ProofWatch | Emits OTLP with full compliance context | `enrichment_status = Success` |
| B — Raw policy engine | OPA, Kyverno, etc. | Emits raw OTLP; `truthbeam` processor enriches | `enrichment_status` varies |
| Local | `cmd/ingest` | Direct ClickHouse insert from YAML files | `enrichment_status = Success` |

The OTel Collector is **environment infrastructure** — Studio does not deploy or manage a collector. See [otel-collector-out-of-chart.md](../decisions/otel-collector-out-of-chart.md).

```bash
# Local development (no collector needed)
CLICKHOUSE_HOST=localhost CLICKHOUSE_PORT=9000 \
  go run ./cmd/ingest path/to/evaluation-log.yaml
```

## Data Flow

### Policy Import

```
OCI Registry → Gateway /api/policies/import → ClickHouse policies table
```

### Evidence Ingestion

```
API POST /api/evidence         ─┐
File upload /api/evidence/upload ├→ Gateway → ClickHouse evidence table
OTel Collector (external)       ─┘
```

### Audit Preparation (Chat)

```
User → Workbench → Gateway /api/a2a/ → Studio Assistant
                                           ├── clickhouse-mcp (query evidence)
                                           └── gemara-mcp (validate artifact)
                                                    │
SSE stream ← Workbench ← Gateway ← Studio Assistant (AuditLog YAML)
```

## Authentication

| Mode | Trigger | Behavior |
|:--|:--|:--|
| Disabled | `GOOGLE_CLIENT_ID` unset | No auth middleware; APIs open |
| Google OAuth | `GOOGLE_CLIENT_ID` set | OpenID Connect flow, AES-GCM encrypted cookie, `/api/*` gated |

## Helm Configuration

Key values in `charts/complytime-studio/values.yaml`:

| Value | Description |
|:--|:--|
| `model.provider` | LLM provider (default: `GeminiVertexAI`) |
| `model.name` | Model identifier (default: `gemini-2.5-pro`) |
| `model.anthropicVertexAI.projectID` | GCP project for Vertex AI |
| `auth.google.clientId` | Google OAuth client ID (enables auth middleware) |
| `clickhouse.enabled` | Deploy ClickHouse evidence store (default: `false`) |
| `registry.enabled` | Deploy in-cluster OCI registry |

## Kubernetes Layout

```
Namespace: kagent
├── Deployments
│   ├── studio-gateway        (Go, embedded A2A proxy)
│   ├── studio-assistant      (Python ADK)
│   ├── studio-gemara-mcp     (managed by KMCP)
│   ├── studio-clickhouse-mcp (managed by KMCP)
│   ├── studio-oras-mcp       (managed by KMCP)
│   └── studio-registry       (Zot, dev only)
├── StatefulSets
│   └── studio-clickhouse     (PVC-backed)
├── ConfigMaps
│   ├── studio-clickhouse-schema (tuning XML + users config)
│   └── studio-agent-prompts     (system prompts)
├── Secrets
│   ├── studio-gcp-credentials
│   ├── studio-clickhouse-credentials
│   └── studio-oauth-credentials (optional)
└── kagent CRDs
    ├── Agent: studio-assistant
    └── MCPServer: gemara-mcp, clickhouse-mcp, oras-mcp
```

## Configuration (Environment Variables)

| Variable | Component | Purpose |
|:--|:--|:--|
| `CLICKHOUSE_ADDR` | Gateway | ClickHouse native protocol address |
| `GEMARA_MCP_URL` | Gateway, Assistant | Gemara MCP HTTP endpoint |
| `CLICKHOUSE_MCP_URL` | Assistant | ClickHouse MCP HTTP endpoint |
| `ORAS_MCP_URL` | Gateway | ORAS MCP HTTP endpoint |
| `A2A_PROXY_URL` | Gateway | External A2A proxy (if not embedded) |
| `AGENT_DIRECTORY` | Gateway | JSON array of agent cards |
| `MODEL_NAME` | Assistant | LLM model identifier |
| `GOOGLE_CLIENT_ID` | Gateway | Enables Google OAuth when set |
| `COOKIE_SECRET` | Gateway | 32-byte hex key for session encryption |
