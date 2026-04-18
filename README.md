# ComplyTime Studio

Multi-agent platform for authoring, validating, and publishing [Gemara](https://gemara.openssf.org/) GRC artifacts as OCI bundles. Specialist agents run as [kagent](https://kagent.dev/) Declarative Agent CRDs on Kubernetes. A thin Go gateway serves the workbench UI, REST endpoints, and an agent directory.

## Quick Start

### Prerequisites

- Kind cluster with kagent >= 0.8.0 and KMCP controller
- GCP Application Default Credentials (for AnthropicVertexAI)
- GitHub personal access token (optional, for github-mcp)

### Kubernetes (kagent)

```bash
make cluster-up       # Kind cluster + kagent operator
make deploy           # Build, load image, install Helm chart, restart gateway
make port-forward     # Forward localhost:8080 → gateway
```

**With model and auth configuration:**

```bash
# Refresh GCP credentials
gcloud auth application-default login
kubectl create secret generic studio-gcp-credentials -n kagent \
  --from-file=application_default_credentials.json=$HOME/.config/gcloud/application_default_credentials.json \
  --dry-run=client -o yaml | kubectl apply -f -

# Set Vertex AI project and optional GitHub OAuth
VERTEX_PROJECT_ID=my-gcp-project \
GITHUB_CLIENT_ID=Ov23li... \
GITHUB_CLIENT_SECRET=abc123 \
  make deploy
```

### Local (Docker Compose)

Runs the gateway and MCP servers. Agents require Kubernetes.

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
├── /api/workspace/save  → Save artifacts to .complytime/artifacts/
├── /auth/*              → GitHub OAuth flow (optional)

kagent Declarative Agents (Go runtime)
├── studio-threat-modeler  — STRIDE analysis, ThreatCatalog + ControlCatalog
│   └── MCP: gemara-mcp, github-mcp
├── studio-gap-analyst     — Evidence-backed AuditLog from ClickHouse L5/L6 data
│   └── MCP: gemara-mcp, github-mcp, clickhouse-mcp
└── studio-policy-composer — RiskCatalog + Policy authoring
    └── MCP: gemara-mcp, github-mcp

MCP Servers (kagent MCPServer CRDs via KMCP)
├── studio-gemara-mcp      — Gemara schema validation + migration (stdio)
├── studio-oras-mcp        — OCI registry operations (stdio)
├── studio-github-mcp      — Repository context (stdio)
└── studio-clickhouse-mcp  — ClickHouse evidence queries (optional, stdio)
```

## Workbench

The workbench is a unified editor-first workspace:

- **Workspace Editor**: CodeMirror YAML editor as the primary view, with toolbar for validate, save, copy, publish, and definition selection
- **Chat Drawer**: Slide-out panel for agent conversations; mission artifacts stream into the editor
- **Validation**: Artifact validation via gemara-mcp with user-selectable definition override
- **Registry Import**: Browse OCI registries, inspect layers, import mapping references into the active editor
- **Agent Picker**: Select specialist agents when starting missions
- **Save to Workspace**: Persist artifacts to `.complytime/artifacts/`
- **Publishing**: Push OCI bundles to registries

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
| `model.name` | Model identifier (`claude-sonnet-4-20250514`) |
| `model.anthropicVertexAI.projectID` | GCP project for Vertex AI |
| `model.anthropicVertexAI.location` | Vertex AI region (default: `us-east5`) |
| `model.anthropicVertexAI.credentialsSecret` | Secret with `application_default_credentials.json` |
| `internalSkills.enabled` | Enable skills from this repo via gitRefs (default: `false`) |
| `auth.github.clientId` | GitHub OAuth client ID (enables auth middleware) |
| `registry.enabled` | Deploy in-cluster OCI registry |

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
| `make port-forward` | Port-forward gateway to localhost:8080 |
| `make oauth-secret` | Create GitHub OAuth credentials secret |
| `make cluster-up` | Create Kind cluster with kagent |
| `make cluster-down` | Delete Kind cluster |
| `make test` | Run Go tests |
| `make lint` | Run golangci-lint |

## License

[Apache License 2.0](LICENSE)
