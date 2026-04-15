# ComplyTime Studio

Multi-agent platform for authoring, validating, and publishing [Gemara](https://gemara.openssf.org/) GRC artifacts as OCI bundles. Specialist agents run as [kagent](https://kagent.dev/) Declarative Agent CRDs on Kubernetes. A thin Go gateway serves the workbench UI, REST endpoints, and an agent directory.

## Quick Start

### Local (Docker Compose)

Runs the gateway and MCP servers. Agents require Kubernetes.

```bash
cp .env.example .env
docker compose up
# Open http://localhost:8080
```

### Kubernetes (kagent)

```bash
make cluster-up       # Kind cluster + kagent operator
make studio-up        # Install Helm chart (Declarative agents, MCP servers, gateway)
kubectl port-forward -n kagent svc/studio-gateway 8080:8080
```

## Architecture

```
Gateway (:8080)
├── /                → Workbench SPA (embedded)
├── /api/agents      → Agent directory (specialist cards)
├── /api/validate    → gemara-mcp proxy
├── /api/migrate     → gemara-mcp proxy
├── /api/registry/*  → oras-mcp proxy
├── /api/publish     → OCI bundle assembly + push

kagent Declarative Agents (Go runtime)
├── studio-threat-modeler  — STRIDE analysis, ThreatCatalog + ControlCatalog
│   └── MCP: gemara-mcp, github-mcp
├── studio-gap-analyst     — Evidence-backed AuditLog from ClickHouse L5/L6 data
│   └── MCP: gemara-mcp, github-mcp, clickhouse-mcp (ClickHouse/mcp-clickhouse)
└── studio-policy-composer — RiskCatalog + Policy authoring
    └── MCP: gemara-mcp, github-mcp

MCP Servers (kagent MCPServer CRDs)
├── studio-gemara-mcp      — Gemara schema validation + migration
├── studio-oras-mcp        — OCI registry operations
├── studio-github-mcp      — Repository context
└── studio-clickhouse-mcp  — ClickHouse evidence queries (optional)
```

## Agent Definitions

Each specialist is defined in `agents/<name>/`:

| File | Purpose |
|:--|:--|
| `agent.yaml` | Name, description, MCP tools, A2A skills, model reference |
| `prompt.md` | System prompt (plain markdown) |

These are framework-agnostic. Helm renders them into kagent CRDs.

## Workbench

- **Chat + Artifacts**: Split-pane view with streaming chat and CodeMirror YAML editor
- **Validation**: Instant artifact validation via gemara-mcp (zero tokens)
- **Publishing**: Push signed OCI bundles to registries
- **Registry Browser**: Discover and pull existing bundles
- **Agent Directory**: Pick specialists directly from the UI

## Build Targets

| Target | Description |
|:--|:--|
| `make gateway-build` | Compile gateway to `bin/studio-gateway` |
| `make gateway-image` | Build gateway container image |
| `make ingest-build` | Compile ingest CLI to `bin/studio-ingest` |
| `make ingest-image` | Build ingest container image |
| `make sync-prompts` | Copy `agents/*/prompt.md` into Helm chart |
| `make compose-up` | Start gateway + MCP servers via docker-compose |
| `make cluster-up` | Create Kind cluster with kagent |
| `make cluster-down` | Delete Kind cluster |
| `make studio-up` | Install Helm chart (syncs prompts first) |
| `make studio-down` | Uninstall Helm chart |
| `make studio-template` | Render Helm templates locally |
| `make workbench-build` | Build workbench SPA |
| `make test` | Run Go tests |
| `make lint` | Run golangci-lint |

## License

[Apache License 2.0](LICENSE)
