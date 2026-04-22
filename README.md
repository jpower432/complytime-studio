# ComplyTime Studio

Compliance analytics and audit preparation dashboard. Aggregates policies from OCI registries and evidence from OTel/API/file upload into a single read-mostly view, then uses an agentic assistant to synthesize [Gemara](https://gemara.openssf.org/) AuditLog artifacts. Artifact authoring (ThreatCatalogs, ControlCatalogs, Policies) stays with engineers in local tooling (Cursor, Claude Code) + gemara-mcp.

Studio is the aggregation point in a decoupled compliance ecosystem — policies live in Git, evidence flows through OTel, and artifacts are distributed via OCI registries. The [Gemara schema](https://gemara.openssf.org/) is the shared contract that ties these together.

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

The agents use LLMs via GCP Vertex AI (default: Claude for specialists, Gemini for the assistant). A GCP project with Vertex AI enabled is **required**.

```bash
gcloud auth application-default login
export VERTEX_PROJECT_ID=my-gcp-project
```

**Google OAuth (optional, enables user authentication):**

```bash
# Create credentials at https://console.cloud.google.com/apis/credentials
# Redirect URI: http://localhost:8080/auth/callback
export GOOGLE_CLIENT_ID=123456...apps.googleusercontent.com
export GOOGLE_CLIENT_SECRET=GOCSPX-...
```

**Admin allowlist (optional, defaults to all-admin):**

Configure `auth.admins` in `charts/complytime-studio/values.yaml` with email addresses. Users not in the list get viewer (read-only) access.

### 3. Create the cluster

```bash
make cluster-up
```

> **Podman users:** The setup script auto-detects podman and patches CoreDNS for rootless networking.

### 4. Build and deploy

```bash
make deploy
```

With Google OAuth:

```bash
GOOGLE_CLIENT_ID=$GOOGLE_CLIENT_ID \
GOOGLE_CLIENT_SECRET=$GOOGLE_CLIENT_SECRET \
  make deploy
```

### 5. Access the workbench

```bash
kubectl port-forward -n kagent svc/studio-gateway 8080:8080
```

Open [http://localhost:8080](http://localhost:8080).

### 6. Seed demo data

```bash
GATEWAY_URL=http://localhost:8080 STUDIO_API_TOKEN=dev-seed-token ./demo/seed.sh
```

Seeds the AMPEL branch protection policy (from [complytime-policies](https://github.com/complytime/complytime-policies)), a SOC 2 mapping document, and 45 evidence records across 3 ComplyTime repositories. See [demo/prompts.md](demo/prompts.md) for a guided walkthrough.

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

## Build Targets

| Target | Description |
|:--|:--|
| `make deploy` | Full build, kind load, helm install, rollout restart |
| `make gateway-build` | Compile gateway to `bin/studio-gateway` |
| `make gateway-image` | Build gateway container image (includes workbench SPA) |
| `make workbench-build` | Build workbench SPA |
| `make sync-prompts` | Copy `agents/*/prompt.md` into Helm chart |
| `make studio-up` | Install/upgrade Helm chart |
| `make studio-down` | Uninstall Helm chart |
| `make oauth-secret` | Create Google OAuth credentials secret |
| `make cluster-up` | Create Kind cluster with kagent |
| `make cluster-down` | Delete Kind cluster |
| `make test` | Run Go tests |
| `make lint` | Run golangci-lint |

## Documentation

| Document | Purpose |
|:--|:--|
| [Architecture](docs/design/architecture.md) | System design, components, data flows |
| [Agent Data Flows](docs/design/agent-data-flows.md) | Workbench-to-agent communication |
| [Evidence Semconv](docs/design/evidence-semconv-alignment.md) | OTel attribute-to-ClickHouse mapping |
| [Decisions](docs/decisions/) | Architecture Decision Records |

## License

[Apache License 2.0](LICENSE)
