# ComplyTime Studio

Audit preparation platform for automated evidence ingestion and compliance analytics. Built around the [OpenSSF Gemara](https://gemara.openssf.org/) project.

Studio ingests evidence from scanning tools, maps it against compliance framework requirements, and uses AI agents to synthesize audit-ready artifacts. Policies and evidence stay in their source systems — Studio aggregates summaries and computes posture.

## What It Does

| Capability | What you get |
|:--|:--|
| **Program Management** | Track compliance programs (SOC 2, FedRAMP, ISO 27001) with attached policies and target environments |
| **Posture Analytics** | See which requirements are covered, which have gaps, and where evidence is stale or missing |
| **Evidence Ingestion** | Ingest evidence from scanning tools via REST API — each record maps to a control and requirement |
| **Requirement Coverage** | View control-by-control coverage with evidence counts, classifications, and drill-down |
| **Audit Preparation** | AI agents draft [Gemara AuditLog](https://gemara.openssf.org/) artifacts; humans review and promote to official records |
| **Notifications** | Activity feed for evidence arrivals, posture changes, and draft audit logs awaiting review |

## Repository Structure

ComplyTime spans four repositories:

| Repository | Role | Language |
|:--|:--|:--|
| **complytime-studio** (this repo) | Data Platform — evidence CRUD, posture, certifier pipeline, auth | Go |
| [studio-ui](https://github.com/complytime/studio-ui) | Batteries-included SPA + Nginx reverse-proxy | TypeScript |
| [complytime-agents](https://github.com/complytime/complytime-agents) | Studio Workbench + AI agents — A2A routing, chat, Gemara tools | Python |
| [studio-deploy](https://github.com/complytime/studio-deploy) | Helm chart + Docker Compose for local/cluster deployment | YAML |

## Quick Start

### Prerequisites

| Tool | Purpose | Install |
|:--|:--|:--|
| `docker` or `podman` | Container runtime | [docker.com](https://docs.docker.com/get-docker/) / `dnf install podman` |
| `kind` | Local Kubernetes cluster | [kind.sigs.k8s.io](https://kind.sigs.k8s.io/docs/user/quick-start/#installation) |
| `kubectl` | Kubernetes CLI | [kubernetes.io](https://kubernetes.io/docs/tasks/tools/) |
| `helm` | Chart management | [helm.sh](https://helm.sh/docs/intro/install/) |
| `go` (>= 1.25) | Build the gateway | [go.dev](https://go.dev/dl/) |

### Deploy (Kubernetes)

See [studio-deploy](https://github.com/complytime/studio-deploy) for Helm chart and Docker Compose orchestration.

### Run gateway locally

```bash
make gateway-build
./bin/studio-gateway
```

### Seed demo data

```bash
make seed
```

## Architecture

```
Browser → Nginx (studio-ui)
            ├── /api/*        → Data Platform (Go gateway)
            ├── /auth/*       → Data Platform
            ├── /workbench/*  → Studio Workbench (Python)
            │                      ├── A2A routing → LangGraph agents
            │                      ├── Gemara validate/migrate (MCP)
            │                      └── OCI publish/browse (MCP)
            └── /*            → static SPA files
```

**Data Platform** (this repo) is a headless data API: evidence CRUD, posture computation, certifier pipeline (NATS), content ingestion, auth. PostgreSQL stores all application data.

**Studio Workbench** ([complytime-agents](https://github.com/complytime/complytime-agents)) serves agent-support endpoints: A2A routing, agent directory, chat state, Gemara validate/migrate, OCI publish/browse. Agents consume platform state through `studio-mcp` MCP resources.

**Studio UI** ([studio-ui](https://github.com/complytime/studio-ui)) is a batteries-included Preact SPA. Nginx routes requests to the correct backend by path prefix.

For full architecture detail, see [`docs/architecture.md`](docs/architecture.md).

## Development

| Target | Description |
|:--|:--|
| `make gateway-build` | Compile gateway to `bin/studio-gateway` |
| `make gateway-image` | Build gateway container image |
| `make studio-mcp-build` | Compile `studio-mcp` to `bin/studio-mcp` |
| `make studio-mcp-image` | Build `studio-mcp` container image |
| `make test` | Run Go tests |
| `make lint` | Run golangci-lint |
| `make seed` | Seed demo data |

Deployment targets (`cluster-up`, `deploy`, `helm-*`) moved to [studio-deploy](https://github.com/complytime/studio-deploy).

## Documentation

| Document | Purpose |
|:--|:--|
| [Architecture](docs/architecture.md) | Component boundaries, routing, communication |
| [Service Level Requirements](docs/requirements/service-level-requirements.md) | SLRs, ownership, gap analysis |
| [Agent Data Flows](docs/design/agent-data-flows.md) | Workbench-to-agent communication |
| [Evidence Semconv](docs/design/evidence-semconv-alignment.md) | Evidence column mapping to OTel semantic conventions |
| studio-mcp | MCP resources and tools for agents (see `cmd/studio-mcp/`) |
| [Decisions](docs/decisions/) | Architecture Decision Records |

## License

[Apache License 2.0](LICENSE)
