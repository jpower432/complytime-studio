## Why

Studio's three MCP servers (gemara-mcp, oras-mcp, github-mcp-server) are stdio-only. The Helm chart deploys them as plain Deployments with HTTP Service frontends, but no protocol bridge exists — the orchestrator's SSE/HTTP connections are refused at startup. The KMCP controller already running in the cluster solves this exact problem via `MCPServer` CRDs with `transportType: stdio`, which inject an AgentGateway sidecar to bridge stdio to Streamable HTTP automatically.

## What Changes

- **Replace** 3 Deployment + 3 Service resources in `mcp-servers.yaml` with 3 `MCPServer` CRDs (`kagent.dev/v1alpha1`).
- **Update** orchestrator env vars (`GEMARA_MCP_URL`, `ORAS_MCP_URL`, `GITHUB_MCP_URL`) to point to KMCP-generated Service endpoints.
- **Remove** `stdin: true` workaround and manual `containerPort: 3000` declarations from MCP server templates.
- **Update** `values.yaml` to reflect MCPServer-specific config (command, args, env) instead of raw Deployment fields.
- **Simplify** the `setup.sh` secret creation for GitHub token to align with MCPServer CRD `secretRefs` pattern.

## Capabilities

### New Capabilities

- `mcpserver-crd-transport`: Replace manual Deployment+Service pairs with kagent MCPServer CRDs that use the KMCP controller's built-in stdio-to-HTTP bridge (AgentGateway sidecar).

### Modified Capabilities

(none — no existing spec-level requirements change)

## Impact

| Area | Detail |
|:---|:---|
| `charts/complytime-studio/templates/mcp-servers.yaml` | Rewritten: 3 MCPServer CRDs replace 6 Deployment+Service resources |
| `charts/complytime-studio/templates/agent-orchestrator.yaml` | MCP URL env vars updated to KMCP-generated service names/ports |
| `charts/complytime-studio/values.yaml` | MCP server config restructured for MCPServer CRD fields |
| `deploy/kind/setup.sh` | GitHub token secret key may need alignment with MCPServer `env` pattern |
| **kagent dependency** | Hard dependency on `kagent-kmcp-controller-manager` for MCP server lifecycle |

### Downsides and Risks

| Risk | Severity | Mitigation |
|:---|:---|:---|
| **Tight coupling to kagent MCPServer API** — if kagent changes the CRD schema, the chart breaks | Medium | Pin kagent CRD version in `Chart.yaml` annotations; the CRD is v1alpha1 and may graduate |
| **Opaque bridge layer** — AgentGateway sidecar config is controller-managed, not directly tunable from the chart | Low | KMCP controller exposes `initContainer` image override and `timeout` fields for tuning |
| **Service naming convention** — KMCP controller generates Service names; must verify naming matches our URL env vars | Low | Spike during implementation: create a test MCPServer and inspect generated resources |
| **Startup latency** — AgentGateway spawns a new stdio process per session, adding 2-8s to first connection | Low | Set `timeout: 30s` on MCPServer spec; the orchestrator already has connect timeouts |
| **Single point of failure** — KMCP controller outage prevents MCP server reconciliation | Low | Same risk as any controller-managed resource; kagent controller is already a dependency |
| **Loss of direct Deployment control** — no direct `replicas`, custom probes, or resource limits visible in our templates | Low | MCPServer CRD supports `replicas`, `resources`, `securityContext`, `sidecars` fields |
