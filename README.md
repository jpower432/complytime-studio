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

## Quick Start

### Prerequisites

| Tool | Purpose | Install |
|:--|:--|:--|
| `docker` or `podman` | Container runtime | [docker.com](https://docs.docker.com/get-docker/) / `dnf install podman` |
| `kind` | Local Kubernetes cluster | [kind.sigs.k8s.io](https://kind.sigs.k8s.io/docs/user/quick-start/#installation) |
| `kubectl` | Kubernetes CLI | [kubernetes.io](https://kubernetes.io/docs/tasks/tools/) |
| `helm` | Chart management | [helm.sh](https://helm.sh/docs/intro/install/) |
| `go` (>= 1.25) | Build the gateway | [go.dev](https://go.dev/dl/) |
| `node` / `npm` | Build the workbench SPA | [nodejs.org](https://nodejs.org/) |

### Deploy

```bash
make cluster-up
make deploy
```

### Access

```bash
kubectl port-forward -n kagent svc/studio-gateway 8080:8080
```

Open [http://localhost:8080](http://localhost:8080).

### Seed demo data

```bash
GATEWAY_URL=http://localhost:8080 STUDIO_API_TOKEN=dev-seed-token ./demo/seed.sh
```

### Tear down

```bash
make cluster-down
```

### Local (Docker Compose)

Runs the gateway, PostgreSQL, NATS, and MCP servers without Kubernetes. Agents are not available in this mode.

```bash
docker compose up
```

## Architecture

```
Browser (Preact SPA)
  |
  v
Gateway (Go)  ──── PostgreSQL (all application data)
  |
  +--> NATS (required) ──── event-driven services (ingest, posture notifications, etc.)
  |
  | A2A
  v
AI Agents (kagent)  ──── MCP tools (gemara-mcp, oras-mcp)
```

**Gateway** serves the workbench SPA, REST APIs, and proxies agent communication. PostgreSQL stores programs, users, evidence, policies, audit logs, and all analytics data.

**Agents** run as [kagent](https://github.com/kagent-dev/kagent) Declarative Agents in Kubernetes. They use MCP tools to validate and publish Gemara artifacts.

**Authentication** supports any OIDC-compliant provider (Keycloak, Okta, Azure AD, Google). Role assignment via JWT claims, bootstrap allowlist, or first-admin promotion.

**Model providers** currently support Vertex AI (Anthropic, Gemini). The architecture is provider-agnostic — additional backends can be added through kagent's model configuration.

For REST API reference, deployment configuration, and data flows, see [Architecture](docs/design/architecture.md).

## Development

| Target | Description |
|:--|:--|
| `make deploy` | Build, load to kind, helm install, rollout restart |
| `make gateway-build` | Compile gateway to `bin/studio-gateway` |
| `make gateway-image` | Build gateway container image (includes workbench) |
| `make workbench-build` | Build workbench SPA |
| `make test` | Run Go tests |
| `make lint` | Run golangci-lint |
| `make seed` | Seed demo data |
| `make cluster-up` | Create kind cluster with kagent |
| `make cluster-down` | Delete kind cluster |
| `make compose-up` | Docker Compose (gateway + PostgreSQL + NATS + MCP, no agents) |

## Documentation

| Document | Purpose |
|:--|:--|
| [Architecture](docs/design/architecture.md) | System design, REST API, deployment, data flows |
| [Agent Data Flows](docs/design/agent-data-flows.md) | Workbench-to-agent communication |
| [Evidence Semconv](docs/design/evidence-semconv-alignment.md) | Evidence column mapping to OTel semantic conventions |
| [Decisions](docs/decisions/) | Architecture Decision Records |

## License

[Apache License 2.0](LICENSE)
