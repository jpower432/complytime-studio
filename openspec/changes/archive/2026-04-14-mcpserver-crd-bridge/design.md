## Context

The chart deploys three MCP servers as standard Kubernetes Deployments + ClusterIP Services. The latest upstream images (`gemara-mcp:latest`, `oras-mcp:main`, `github-mcp-server:latest`) only support stdio transport. The BYO agent connects to MCP servers over HTTP via `StreamableClientTransport`. No protocol bridge exists, so the orchestrator logs "connection refused" warnings and disables gemara-mcp proxy and registry proxy functionality at startup.

The kagent KMCP controller (`kagent-kmcp-controller-manager`) is already deployed in the cluster. It reconciles `MCPServer` CRDs and, for `transportType: stdio`, injects an AgentGateway init container that copies a transport adapter binary into the pod. This adapter spawns the stdio MCP process and exposes it as a Streamable HTTP endpoint.

**Current resource flow (broken):**

```
chart template  →  Deployment + Service  →  Pod (stdio-only container)
                                              ↑
                                    agent HTTP request → connection refused
```

**Target resource flow:**

```
chart template  →  MCPServer CRD  →  KMCP controller reconciles:
                                       → Deployment (MCP container + AgentGateway sidecar)
                                       → Service (HTTP endpoint)
                                       → RemoteMCPServer (discovery)
                                              ↑
                                    agent HTTP request → AgentGateway → stdio process
```

## Goals / Non-Goals

**Goals:**

- Establish working HTTP connectivity between the BYO agent and all three MCP servers in the Kind cluster.
- Reduce chart template complexity by delegating deployment lifecycle to the KMCP controller.
- Use the same MCPServer CRD pattern that kagent's ecosystem expects, so Studio MCP servers are visible in kagent's UI and API.

**Non-Goals:**

- Modifying the Go agent code's transport logic (`internal/agents/config.go`). The agent already supports `MCP_TRANSPORT=sse` and URL-based connections — those remain unchanged.
- Supporting direct SSE transport in the MCP server images. That's an upstream concern.
- Production hardening (TLS, mTLS, horizontal scaling). This change targets the dev Kind cluster. Production overlays are future work.
- Migrating the OCI registry or seed job to MCPServer CRDs — those are not MCP servers.

## Decisions

### 1. Use `MCPServer` CRD with `transportType: stdio` (not manual AgentGateway sidecar)

**Choice:** Let the KMCP controller manage the AgentGateway injection and Service creation.

**Alternatives considered:**
- *Manual sidecar in our Deployment templates* — more control, but duplicates what KMCP already does. Violates "reuse existing infrastructure" principle. Requires tracking AgentGateway image versions ourselves.
- *supergateway (npm) or mcp-proxy (Python)* — adds a non-Go dependency. Requires building custom bridge images. No Kubernetes-native lifecycle management.

**Rationale:** KMCP is installed, running, and purpose-built for this. One dependency path is better than two.

### 2. MCPServer naming convention: `studio-gemara-mcp`, `studio-oras-mcp`, `studio-github-mcp`

**Choice:** Keep the same resource names as the existing Deployments/Services.

**Rationale:** The KMCP controller creates a Service with the same name as the MCPServer resource. By reusing the current names, the orchestrator's URL env vars (`http://studio-gemara-mcp:<port>`) require only a port update (if the controller uses a different default). Must verify the generated Service port during implementation.

### 3. MCP server config expressed via MCPServer `deployment` spec fields

**Choice:** Use `cmd`, `args`, and `env` fields in the MCPServer `deployment` spec. Use `secretRefs` for the GitHub token.

**Rationale:** The MCPServer CRD's deployment spec supports all fields we need. `secretRefs` mounts secrets as volumes automatically, but the GitHub MCP server needs the token as an env var — use `env` with a literal value referencing the secret via the setup script's existing secret creation.

### 4. Orchestrator URL env vars point to KMCP-generated Services

**Choice:** Update the orchestrator template's `GEMARA_MCP_URL`, `ORAS_MCP_URL`, and `GITHUB_MCP_URL` to use the port and path the KMCP controller exposes.

**Depends on:** Verifying the KMCP-generated Service port and MCP path. The MCPServer deployment spec has a `port` field (default 3000) and `httpTransport.path` can specify the MCP endpoint path. The agent uses `StreamableClientTransport{Endpoint: url}` which should match.

## Risks / Trade-offs

| Risk | Mitigation |
|:---|:---|
| KMCP controller generates Service with unexpected port or path → orchestrator still gets "connection refused" | Create a test MCPServer first, inspect generated Service and RemoteMCPServer, then wire the URL |
| AgentGateway init container image not available or pull-restricted in air-gapped environments | MCPServer CRD supports `initContainer.image` override; document this in values.yaml comments |
| `MCPServer` CRD is `v1alpha1` — breaking changes possible in kagent upgrades | Pin kagent Helm chart version in `setup.sh`; test upgrades in CI before adopting |
| GitHub MCP server needs `GITHUB_PERSONAL_ACCESS_TOKEN` as env var; MCPServer `secretRefs` mounts as volume, not env | Use `env` field in MCPServer deployment spec with explicit key/value, or pass the secret name and have the setup script align the key |
| Loss of chart-level label consistency — KMCP controller sets its own labels on generated Deployments | MCPServer deployment spec supports `labels` and `annotations` fields; pass our standard labels |

## Open Questions

1. **What port does KMCP expose?** The MCPServer `deployment.port` defaults to 3000. Does the generated Service also use 3000, or does AgentGateway listen on a different port?
2. **What path does AgentGateway serve MCP on?** Is it `/mcp`, `/`, or configurable via `httpTransport.path`? The orchestrator currently points to the root URL.
3. **Does the KMCP controller tolerate the github-mcp-server's `stdio` subcommand?** The github-mcp-server binary uses `stdio` as a positional subcommand (not a flag). Need to verify `cmd` + `args` pass through correctly.
