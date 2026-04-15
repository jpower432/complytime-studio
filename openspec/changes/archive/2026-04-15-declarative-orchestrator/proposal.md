## Why

The Studio orchestrator is a routing layer — it decides *who* does work, not *how* work is done. Today it's a BYO Agent CRD running a custom Go binary that also serves the workbench SPA, REST proxy endpoints, and the publish pipeline. This conflates platform integration (skills, observability, model config) with web serving and agent logic in a single binary.

Making the orchestrator a kagent Declarative Agent unlocks Git-based skill loading (prompts as `SKILL.md` files, loaded on-demand via `load_skill`), OpenTelemetry tracing across multi-agent delegations, prompt auditing for compliance, and shared `ModelConfig` CRDs — none of which are available to BYO agents. The specialist agents remain BYO to preserve runtime portability (local dev, CI, Docker, non-Kubernetes environments).

## What Changes

- **Orchestrator becomes a Declarative Agent CRD** (`type: Declarative`, `runtime: go`) with `systemMessage`, `modelConfig`, and tool references replacing the current Go ADK wiring in `internal/agents/orchestrator.go`.
- **Specialists referenced as agent-tools** — the orchestrator's `tools` array uses `type: Agent` entries pointing at the BYO threat modeler, gap analyst, and policy composer Agent CRDs.
- **MCP servers referenced as McpServer tools** — oras-mcp and github-mcp wired via `type: McpServer` in the orchestrator's tool list instead of Go code.
- **New `studio-gateway` service** — a thin Go binary serving the workbench SPA, REST proxy endpoints (`/api/validate`, `/api/migrate`, `/api/registry/*`), the publish endpoint (`/api/publish`), and proxying A2A requests to the orchestrator.
- **Specialist agents deployed as separate BYO Agent CRDs** — each specialist gets its own Agent CRD and pod (today they run as goroutines inside the orchestrator pod).
- **Orchestrator prompt moves to Git-based skill** — `orchestrator_prompt.md` becomes a `SKILL.md` file loadable via `load_skill`, enabling prompt iteration without image rebuilds.
- **Shared `ModelConfig` CRD** — single model configuration referenced by the declarative orchestrator, replacing per-agent env var wiring.

## Capabilities

### New Capabilities
- `declarative-orchestrator`: Orchestrator agent expressed as a kagent Declarative Agent CRD with tool references to specialist agents and MCP servers.
- `studio-gateway`: Thin web service serving the workbench SPA, REST proxy to MCP servers, publish endpoint, and A2A proxy to the orchestrator.
- `specialist-agent-crds`: Each specialist (threat modeler, gap analyst, policy composer) deployed as an independent BYO Agent CRD with its own pod.

### Modified Capabilities
- `mcpserver-crd-transport`: MCP servers now also referenced as `type: McpServer` tools in the declarative orchestrator spec (in addition to existing MCPServer CRDs).

## Impact

- **Code**: `cmd/agents/main.go` splits into `cmd/gateway/main.go` (web layer) and is removed as the orchestrator entry point. `internal/agents/orchestrator.go` is removed (replaced by CRD). Specialist agent constructors (`threatmodeler.go`, `gap_analyst.go`, `policy_composer.go`) move to standalone binaries or remain as-is if the single binary serves all three.
- **Helm chart**: New templates for `studio-gateway` Deployment/Service, declarative orchestrator Agent CRD, `ModelConfig` CRD, and three BYO specialist Agent CRDs. Current `agent-orchestrator.yaml` is replaced.
- **Images**: New `studio-gateway` image. Existing `studio-agents` image continues for specialists. No orchestrator image needed (kagent provides the runtime).
- **Dependencies**: kagent v0.8.1+ required (Go runtime remote-agents fix, PR #1538, merged April 8 2026).
- **Deployment topology**: One pod today → five pods minimum (gateway, orchestrator, 3 specialists). Higher resource baseline but independent scaling and failure isolation.
- **Breaking**: The single `studio-agents:local` image no longer serves the workbench or orchestrator. Users running `make studio-up` get a multi-pod deployment.
