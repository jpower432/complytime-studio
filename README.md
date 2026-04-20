# ComplyTime Studio

Multi-agent platform for authoring, validating, and publishing [Gemara](https://gemara.openssf.org/) GRC artifacts as OCI bundles. Specialist agents run as [kagent](https://kagent.dev/) Declarative Agent CRDs on Kubernetes. A thin Go gateway serves the workbench UI, REST endpoints, and an agent directory.

## Quick Start

### 1. Install prerequisites

| Tool | Purpose | Install |
|:-----|:--------|:--------|
| `docker` or `podman` | Container runtime | [docker.com](https://docs.docker.com/get-docker/) / `dnf install podman` |
| `kind` | Local Kubernetes cluster | [kind.sigs.k8s.io](https://kind.sigs.k8s.io/docs/user/quick-start/#installation) |
| `kubectl` | Kubernetes CLI | [kubernetes.io](https://kubernetes.io/docs/tasks/tools/) |
| `helm` | Chart management | [helm.sh](https://helm.sh/docs/intro/install/) |
| `go` (>= 1.22) | Build the gateway | [go.dev](https://go.dev/dl/) |
| `node` / `npm` | Build the workbench SPA | [nodejs.org](https://nodejs.org/) |
| `gcloud` | GCP credentials for Vertex AI | [cloud.google.com](https://cloud.google.com/sdk/docs/install) |

### 2. Configure credentials

The agents use Anthropic Claude via GCP Vertex AI. Both values are **required**.

```bash
# Authenticate with GCP (creates ~/.config/gcloud/application_default_credentials.json)
gcloud auth application-default login

# Export your GCP project ID (required — cluster setup will fail without it)
export VERTEX_PROJECT_ID=my-gcp-project
```

**GitHub OAuth (required for agent GitHub access):**

Agents access GitHub repositories using the logged-in user's OAuth token propagated via `allowedHeaders`. A static GitHub PAT is also required for MCP session initialization (tool discovery), which occurs before per-request headers are available.

```bash
# Register an OAuth app at https://github.com/settings/applications/new
# Set the callback URL to http://localhost:8080/auth/callback
export GITHUB_CLIENT_ID=Ov23li...
export GITHUB_CLIENT_SECRET=abc123...
```

> **Note:** The OAuth app must request `repo` scope (configured automatically by the gateway) for agents to access private repositories.

**GitHub MCP static token (required):**

The GitHub MCP server needs a static PAT for MCP session initialization. Tool calls use the user's OBO token when available.

```bash
kubectl create secret generic studio-github-token -n kagent \
  --from-literal=GITHUB_PERSONAL_ACCESS_TOKEN=ghp_...
```

> **Why both?** The kagent Python runtime initializes MCP sessions (tool discovery) before per-request headers are available. The static token authenticates session init; `allowedHeaders` propagates the user's OAuth token for actual tool calls.

### 3. Create the cluster

Creates a Kind cluster, installs kagent CRDs and operator, and configures GCP + GitHub secrets.

```bash
make cluster-up
```

> **Podman users:** The setup script auto-detects podman and patches CoreDNS for rootless networking. No manual steps needed.

### 4. Build and deploy

Builds the gateway image (Go binary + workbench SPA), loads it into Kind, installs the Helm chart, and restarts the gateway pod.

```bash
make deploy
```

With GitHub OAuth (enables login):

```bash
GITHUB_CLIENT_ID=$GITHUB_CLIENT_ID \
GITHUB_CLIENT_SECRET=$GITHUB_CLIENT_SECRET \
  make deploy
```

### 5. Access the workbench

```bash
kubectl port-forward -n kagent svc/studio-gateway 8080:8080
```

Open [http://localhost:8080](http://localhost:8080).

### Tear down

```bash
make cluster-down
```

### Local (Docker Compose)

Runs the gateway and MCP servers without Kubernetes. Agents are not available in this mode.

```bash
cp .env.example .env
docker compose up
# Open http://localhost:8080
```

## Architecture

```
Gateway (:8080)
├── /                    → Workbench SPA (embedded)
├── /api/agents          → Agent directory (specialist cards)
├── /api/a2a/{agent}     → A2A reverse proxy → agent pod /invoke
├── /api/validate        → gemara-mcp proxy
├── /api/migrate         → gemara-mcp proxy
├── /api/registry/*      → oras-mcp proxy (or direct OCI API for insecure registries)
├── /api/publish         → OCI bundle assembly + push
├── /api/config          → Platform configuration (GitHub org/repo)
├── /auth/*              → GitHub OAuth flow (optional)

kagent Declarative Agents (Python runtime)
├── studio-threat-modeler  — STRIDE analysis, ThreatCatalog + ControlCatalog
│   └── MCP: gemara-mcp, github-mcp
├── studio-gap-analyst     — Evidence-backed AuditLog from ClickHouse L5/L6 data
│   └── MCP: gemara-mcp, github-mcp, clickhouse-mcp
└── studio-policy-composer — RiskCatalog + Policy authoring
    └── MCP: gemara-mcp, github-mcp

MCP Servers (kagent MCPServer CRDs via KMCP)
├── studio-gemara-mcp      — Gemara schema validation + migration (stdio)
├── studio-oras-mcp        — OCI registry operations (stdio)
├── studio-github-mcp      — Repository context (http, OBO)
└── studio-clickhouse-mcp  — ClickHouse evidence queries (optional, stdio)
```

## Workbench

The workbench is a unified editor-first workspace:

- **Workspace Editor**: CodeMirror YAML editor as the primary view, with toolbar for validate, download, copy, publish, and definition selection
- **Chat Drawer**: Slide-out panel for agent conversations; job artifacts stream into the editor
- **Validation**: Artifact validation via gemara-mcp with user-selectable definition override
- **Registry Import**: Browse OCI registries, inspect layers, import mapping references into the active editor
- **Agent Picker**: Select specialist agents when starting jobs
- **Download YAML**: Export artifacts as YAML files for local Git workflows
- **Publishing**: Push OCI bundles to registries
- **Theme Toggle**: Dark/light mode with system preference detection

## Agent Definitions

Each specialist is defined in `agents/<name>/`:

| File | Purpose |
|:--|:--|
| `agent.yaml` | Name, description, MCP tools, A2A skills, model reference |
| `prompt.md` | System prompt (plain markdown) |

Helm renders these into kagent Agent CRDs. The `ModelConfig` CRD configures the LLM provider (AnthropicVertexAI) with GCP credentials injected via `apiKeySecret`.

## Helm Configuration

Key values in `charts/complytime-studio/values.yaml`:

| Value | Description |
|:--|:--|
| `model.provider` | LLM provider (`AnthropicVertexAI`) |
| `model.name` | Model identifier (`claude-sonnet-4`) |
| `model.anthropicVertexAI.projectID` | GCP project for Vertex AI |
| `model.anthropicVertexAI.location` | Vertex AI region (default: `us-east5`) |
| `model.anthropicVertexAI.credentialsSecret` | Secret with `application_default_credentials.json` |
| `internalSkills.enabled` | Enable skills from this repo via gitRefs (default: `false`) |
| `auth.github.clientId` | GitHub OAuth client ID (enables auth middleware) |
| `github.org` | GitHub organization/user for platform links |
| `github.repo` | GitHub repository name |
| `mcpServers.github.tokenSecret` | Secret name with `GITHUB_PERSONAL_ACCESS_TOKEN` for MCP session init |
| `clickhouse.enabled` | Deploy ClickHouse evidence store (default: `false`) |
| `registry.enabled` | Deploy in-cluster OCI registry |

## Evidence Pipeline

The evidence pipeline ingests compliance assessment data into ClickHouse for the gap-analyst. Two intake paths feed a single `evidence` table.

### Table Schema

All evidence — evaluations and remediations — is stored in a single `evidence` table. Remediation columns are nullable for evaluation-only records. Schema defined in `charts/complytime-studio/templates/clickhouse-schema-configmap.yaml`.

### Ingestion Paths

| Path | Source | How | Enrichment |
|:-----|:-------|:----|:-----------|
| A — Gemara-native | `complyctl` via ProofWatch | Emits OTLP with full compliance context | `enrichment_status = Success` |
| B — Raw policy engine | OPA, Kyverno, etc. | Emits raw OTLP; `truthbeam` processor enriches | `enrichment_status` varies |
| Local | `cmd/ingest` | Direct ClickHouse insert from YAML files | `enrichment_status = Success` |

### Collector Topologies

The OTel Collector is **environment infrastructure**, not an application component. Studio does not deploy or manage a collector — the cluster operator provisions it alongside other observability tooling. See `docs/decisions/otel-collector-out-of-chart.md` for rationale.

Common topologies:

- **Central gateway:** A shared OTel Collector receives OTLP from all producers and exports to ClickHouse.
- **Agent sidecar:** Co-located collector in the same pod as the policy engine. Exporter points at the central ClickHouse instance.
- **Direct (local development):** Use `cmd/ingest` to insert evidence directly from Gemara YAML files without any collector:

```bash
CLICKHOUSE_HOST=localhost CLICKHOUSE_PORT=9000 \
  go run ./cmd/ingest path/to/evaluation-log.yaml
```

### Semantic Convention Alignment

Evidence attributes follow the `beacon.evidence` OTel semantic convention from [complytime-collector-components](https://github.com/complytime/complytime-collector-components). The full attribute-to-column mapping is in `docs/design/evidence-semconv-alignment.md`.

## Build Targets

| Target | Description |
|:--|:--|
| `make deploy` | Full build → kind load → helm install → restart cycle |
| `make gateway-build` | Compile gateway to `bin/studio-gateway` |
| `make gateway-image` | Build gateway container image (includes workbench SPA) |
| `make workbench-build` | Build workbench SPA |
| `make sync-prompts` | Copy `agents/*/prompt.md` into Helm chart |
| `make studio-up` | Install/upgrade Helm chart |
| `make studio-down` | Uninstall Helm chart |
| `make oauth-secret` | Create GitHub OAuth credentials secret |
| `make cluster-up` | Create Kind cluster with kagent |
| `make cluster-down` | Delete Kind cluster |
| `make test` | Run Go tests |
| `make lint` | Run golangci-lint |

## License

[Apache License 2.0](LICENSE)
