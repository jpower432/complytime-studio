# ComplyTime Studio

Compliance analytics and audit preparation platform for compliance analysts and auditors. Aggregates policies from OCI registries and evidence from OTel/REST/CSV into structured views, then uses an agentic assistant to synthesize [Gemara](https://gemara.openssf.org/) AuditLog artifacts.

Studio is the aggregation point in a decoupled compliance ecosystem -- policies live in Git, evidence flows through OTel, raw artifacts stay in per-boundary OCI registries, and Studio holds **summaries only**. The [Gemara schema](https://gemara.openssf.org/) is the shared contract.

## What It Does

| Capability | Description |
|:--|:--|
| **Compliance Posture** | Per-policy cards with pass/fail/other counts, pass rate, target/control inventory, evidence freshness, RACI-style owner, optional risk severity overlay |
| **Posture Drill-down** | Breadcrumb + tabbed **Requirements** / **Evidence** / **History** per policy (`#posture/{id}`) |
| **Requirement Matrix** | Control → requirement → evidence with classifications (**No Evidence** replaces legacy "Blind"), filters, severity context |
| **Inbox** | Notifications from posture/evidence events (when NATS enabled) plus draft audit logs awaiting promotion |
| **Evidence Management** | OTel, REST JSON, **multipart** JSON + files (with blob store), CSV upload, manual entry — full semconv alignment |
| **Audit Export** | CSV, **Excel** (matrix + evidence inventory), **PDF** — scoped by policy and audit window (row caps apply) |
| **Audit History** | AuditLog artifacts with period-over-period comparison |
| **Agent Assistant** | Chat overlay with canned queries, context injection, sticky notes; produces validated AuditLog YAML |

## Quick Start

### 1. Prerequisites

| Tool | Purpose | Install |
|:--|:--|:--|
| `docker` or `podman` | Container runtime | [docker.com](https://docs.docker.com/get-docker/) / `dnf install podman` |
| `kind` | Local Kubernetes cluster | [kind.sigs.k8s.io](https://kind.sigs.k8s.io/docs/user/quick-start/#installation) |
| `kubectl` | Kubernetes CLI | [kubernetes.io](https://kubernetes.io/docs/tasks/tools/) |
| `helm` | Chart management | [helm.sh](https://helm.sh/docs/intro/install/) |
| `go` (>= 1.22) | Build the gateway | [go.dev](https://go.dev/dl/) |
| `node` / `npm` | Build the workbench SPA | [nodejs.org](https://nodejs.org/) |
| `gcloud` | GCP credentials for Vertex AI | [cloud.google.com](https://cloud.google.com/sdk/docs/install) |

### 2. Configure credentials

```bash
gcloud auth application-default login
export VERTEX_PROJECT_ID=my-gcp-project
```

**Google OAuth (optional):**

```bash
export GOOGLE_CLIENT_ID=123456...apps.googleusercontent.com
export GOOGLE_CLIENT_SECRET=your-client-secret
```

### 3. Create cluster and deploy

```bash
make cluster-up
make deploy
```

With Google OAuth:

```bash
GOOGLE_CLIENT_ID=$GOOGLE_CLIENT_ID \
GOOGLE_CLIENT_SECRET=$GOOGLE_CLIENT_SECRET \
  make deploy
```

### 4. Access the workbench

```bash
kubectl port-forward -n kagent svc/studio-gateway 8080:8080
```

Open [http://localhost:8080](http://localhost:8080).

### 5. Seed demo data

```bash
GATEWAY_URL=http://localhost:8080 STUDIO_API_TOKEN=dev-seed-token ./demo/seed.sh
```

Seeds the AMPEL branch protection policy, a SOC 2 mapping document, and 45 evidence records. See [demo/prompts.md](demo/prompts.md) for a guided walkthrough.

### Tear down

```bash
make cluster-down
```

### Local (Docker Compose)

Runs the gateway and MCP servers without Kubernetes. Agents are not available in this mode.

```bash
cp .env.example .env
docker compose up
```

## Architecture

```
Browser (Preact SPA)
  |
  | HTTP / SSE
  v
Gateway public (Go :8080)  --- REST ---> ClickHouse
  |                          --- S3 ---> Blob storage (optional; MinIO-compatible)
  |                          --- optional NATS ---> posture checks / inbox
  |
  +-- Gateway internal (:8081, /internal/* only, NetworkPolicy)
  |
  | A2A proxy
  v
Studio Assistant (Python ADK)  ---> internal gateway URL for draft creation
  |
  | MCP tools
  v
gemara-mcp / clickhouse-mcp / oras-mcp
```

**Key design decisions:**

- **Dashboard-first.** Structured views (posture, requirement matrix, evidence, inbox) are the primary analyst surface. The chat agent augments with synthesis and deep questions.
- **Dual listener.** Public **8080** serves the SPA and `/api/*`. **8081** serves unauthenticated `/internal/*` for trusted workloads — isolate with `NetworkPolicy`. See `docs/decisions/internal-endpoint-isolation.md`.
- **Optional NATS.** Helm `nats.enabled` deploys a single-node NATS bus; `cmd/ingest` publishes evidence events and the gateway debounces posture recomputation and inbox writes. Without `NATS_URL`, core APIs still run.
- **Summary-only ingestion.** Raw evidence never enters Studio. It stays in per-boundary OCI registries as attestation bundles. Studio stores `attestation_ref` + `source_registry` references.
- **Semconv-aligned evidence.** The `evidence` table maps to OTel semantic conventions (`beacon.evidence`). REST, CSV, multipart, and OTel paths write the same columns.
- **Push-only.** Studio does not reach into trust boundaries. `complyctl` pushes summaries; collectors push OTel.

## REST API

| Method | Path | Description |
|:--|:--|:--|
| `GET` | `/api/posture` | Per-policy compliance posture aggregates |
| `GET` | `/api/risks/severity` | Risk severity rows for policy scope |
| `GET` | `/api/requirements` | Requirement matrix with evidence counts |
| `GET` | `/api/requirements/{id}/evidence` | Evidence drill-down for a requirement |
| `GET` | `/api/evidence` | Query evidence with filters |
| `POST` | `/api/evidence` | Ingest evidence (JSON or multipart + files when blob store configured) |
| `POST` | `/api/evidence/upload` | Multipart CSV/JSON file upload |
| `GET` | `/api/notifications` | List inbox notifications |
| `GET` | `/api/notifications/unread-count` | Unread count |
| `PATCH` | `/api/notifications/{id}/read` | Mark read |
| `GET` | `/api/export/csv` | CSV export scoped by policy + audit window |
| `GET` | `/api/export/excel` | Excel export |
| `GET` | `/api/export/pdf` | PDF export |
| `GET` | `/api/policies` | List policies |
| `POST` | `/api/policies/import` | Import policy from OCI |
| `GET` | `/api/audit-logs` | List audit logs |
| `POST` | `/api/audit-logs` | Create audit log |
| `GET` | `/api/draft-audit-logs` | List draft audit logs |
| `PATCH` | `/api/draft-audit-logs/{id}` | Update draft reviewer edits |
| `POST` | `/api/audit-logs/promote` | Promote draft to final |

See [Architecture](docs/design/architecture.md) for the complete API reference.

## Build Targets

| Target | Description |
|:--|:--|
| `make deploy` | Full build, kind load, helm install, rollout restart |
| `make gateway-build` | Compile gateway to `bin/studio-gateway` |
| `make gateway-image` | Build gateway container image (includes workbench SPA) |
| `make workbench-build` | Build workbench SPA |
| `make assistant-image` | Build assistant container image |
| `make sync-prompts` | Copy `agents/*/prompt.md` into Helm chart |
| `make sync-skills` | Copy skills into assistant image |
| `make studio-up` | Install/upgrade Helm chart |
| `make studio-down` | Uninstall Helm chart |
| `make oauth-secret` | Create Google OAuth credentials secret |
| `make cluster-up` | Create Kind cluster with kagent |
| `make cluster-down` | Delete Kind cluster |
| `make compose-up` | Docker Compose (no agents) |
| `make test` | Run Go tests |
| `make lint` | Run golangci-lint |
| `make seed` | Seed demo data |

## Documentation

| Document | Purpose |
|:--|:--|
| [Architecture](docs/design/architecture.md) | System design, components, REST API, data flows |
| [Agent Data Flows](docs/design/agent-data-flows.md) | Workbench-to-agent communication patterns |
| [Evidence Semconv](docs/design/evidence-semconv-alignment.md) | OTel attribute -> ClickHouse column mapping |
| [Decisions](docs/decisions/) | Architecture Decision Records |

**Key ADRs:**

| ADR | Status |
|:--|:--|
| [Cloud-Native Posture Correction](docs/decisions/cloud-native-posture-correction.md) | Proposed |
| [Enforcement Log Traceability](docs/decisions/enforcement-log-traceability.md) | Exploratory |
| [Internal Endpoint Isolation](docs/decisions/internal-endpoint-isolation.md) | Accepted |
| [Query Limit Cap](docs/decisions/query-limit-cap.md) | Accepted |
| [OTel Native Ingestion](docs/decisions/otel-native-ingestion.md) | Accepted |
| [Audit Dashboard Pivot](docs/decisions/audit-dashboard-pivot.md) | Accepted |

## OpenSpec Changes

Active feature specifications in `openspec/changes/`:

| Change | Status |
|:--|:--|
| `sovereignty-model` | Implemented (source_registry column, skill update) |
| `requirement-matrix-view` | Implemented (REST endpoints, workbench view) |
| `manual-evidence-enrichment` | Implemented (multipart, MinIO blob store, validation) |
| `auditor-export` | Implemented (CSV, Excel, PDF) |
| `agent-dashboard-integration` | Partially implemented (posture API, canned queries, real-time updates) |

## License

[Apache License 2.0](LICENSE)
