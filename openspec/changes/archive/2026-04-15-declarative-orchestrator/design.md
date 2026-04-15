## Context

ComplyTime Studio currently runs as a single BYO Agent CRD (`studio-orchestrator`) backed by a monolithic Go binary. This binary contains the orchestrator agent, three specialist agents (threat modeler, gap analyst, policy composer), the workbench SPA, REST proxy endpoints, and the OCI publish pipeline — all in one pod on port 8080.

The orchestrator's Go code (`internal/agents/orchestrator.go`) wires up oras-mcp tools with a filter, a custom `publish_bundle` toolset, and sub-agents as in-process Go ADK `llmagent` instances. The specialists connect to MCP servers (gemara-mcp, oras-mcp, github-mcp) via HTTP or stdio depending on `MCP_TRANSPORT` config.

kagent v0.8.1 introduced Go runtime support for remote agent tools (PR #1538, merged April 8 2026), fixing a blocker where `type: Agent` tool references were silently ignored. This unblocks declarative orchestration over BYO specialists via A2A.

**Constraints:**
- Specialist agents must remain runtime-portable (local dev via `go run`, Docker, CI — no Kubernetes dependency).
- The workbench SPA requires a same-origin backend to proxy requests to cluster-internal MCP services (browser can't reach K8s Services directly).
- kagent Declarative agents use kagent's own runtime image — custom HTTP handlers, embedded SPAs, and arbitrary Go code cannot run inside the declarative pod.

## Goals / Non-Goals

**Goals:**
- Orchestrator expressed as a kagent Declarative Agent CRD with `runtime: go`
- Specialist agents delegated to via `type: Agent` tool references (A2A protocol)
- MCP servers accessible to the orchestrator via `type: McpServer` tool references
- Git-based skill loading for orchestrator prompts via `skills.gitRefs`
- OpenTelemetry tracing and prompt auditing for the orchestrator via kagent's OTel integration
- Shared `ModelConfig` CRD for LLM provider configuration
- Workbench SPA and REST proxy served by a dedicated `studio-gateway` service
- Specialist agents remain BYO with unchanged Go ADK binaries, preserving stdio/sse transport switching

**Non-Goals:**
- Converting specialist agents to Declarative (breaks runtime portability)
- Implementing kagent's workflow sub-agents (Sequential/Parallel/Loop) — the orchestrator's routing is conditional and LLM-driven, not a fixed pipeline
- Adding new specialist agents or capabilities — this change is structural
- Replacing the workbench SPA or changing its feature set
- Modifying the publish pipeline logic (it moves to the gateway, but the code stays the same)

## Decisions

### 1. Orchestrator: Declarative with Python runtime

The orchestrator becomes `type: Declarative` with `runtime: python`. The spike (Phase 1) revealed that the kagent Go runtime does not correctly handle `authorized_user` Application Default Credentials for the AnthropicVertexAI provider, returning `401 CREDENTIALS_MISSING`. The Python runtime uses the Anthropic Python SDK which supports ADC natively.

**Alternative considered:** Keep orchestrator as BYO, add OTel manually. Rejected — would require implementing skill loading, tracing integration, and prompt auditing from scratch, duplicating what kagent provides natively.

**Alternative considered:** Use `runtime: go`. Blocked — kagent Go runtime fails to authenticate with Anthropic Vertex AI using `authorized_user` ADC credentials. Revisit when kagent adds ADC support to the Go runtime's AnthropicVertexAI model implementation.

### 2. Specialists: BYO with separate pods

Each specialist becomes its own BYO Agent CRD and pod. The existing `studio-agents` binary can either run as three separate processes (one per specialist) or remain a single binary with a mode flag.

**Decision: Single binary with `AGENT_MODE` env var.** The binary already has constructors for each specialist (`NewThreatModeler`, `NewGapAnalyst`, `NewPolicyComposer`). Adding a mode flag avoids maintaining three separate binaries while preserving the ability to `go run ./cmd/agents` locally with all agents in one process.

**Alternative considered:** Three separate binaries. Rejected — increases build complexity for marginal benefit. The mode flag approach means `AGENT_MODE=threat-modeler` starts only that agent, while `AGENT_MODE=all` (default) starts all three for local dev.

### 3. Gateway: Thin Go service

A new `cmd/gateway/main.go` binary extracts the non-agent HTTP handlers from the current `cmd/agents/main.go`:
- Workbench SPA serving (embedded Vite assets)
- REST proxy to gemara-mcp (`/api/validate`, `/api/migrate`)
- REST proxy to oras-mcp (`/api/registry/*`)
- Publish endpoint (`/api/publish`)
- A2A proxy to the orchestrator (`/invoke`, `/.well-known/*`)

The gateway is a standard Kubernetes Deployment + Service. It's the user-facing entry point — `kubectl port-forward` targets the gateway, not the orchestrator.

**Alternative considered:** Embed the workbench in the declarative orchestrator. Not possible — declarative agents run kagent's runtime binary, not custom code.

**Alternative considered:** Let the browser call services directly. Rejected — requires Ingress per service, CORS configuration, and exposes internal services to the browser.

### 4. Publish tool: Dual availability

The `publish_bundle` tool needs to be available to both:
- The orchestrator (as an ADK tool for LLM-driven publishing)
- The gateway (as a REST endpoint for workbench UI publishing)

**Decision: Keep publish as a REST endpoint on the gateway. The orchestrator calls it via an MCP-wrapped HTTP tool or a dedicated `publish-mcp` McpServer.**

Simplest path: the orchestrator's system prompt instructs it to return artifacts to the user for publishing via the workbench UI, removing the need for the orchestrator to call publish directly. The publish tool in the orchestrator was always a convenience — the user can click "Publish" in the workbench after reviewing artifacts.

If direct orchestrator publishing is needed later, wrap the gateway's `/api/publish` as a lightweight MCP server.

### 5. ModelConfig CRD

A single `ModelConfig` resource replaces per-agent environment variables:

```yaml
apiVersion: kagent.dev/v1alpha2
kind: ModelConfig
metadata:
  name: studio-model-config
spec:
  provider: AnthropicVertexAI
  model: claude-sonnet-4@20250514
  anthropicVertexAI:
    projectID: "{{ .Values.agents.vertexAI.projectID }}"
    location: "{{ .Values.agents.vertexAI.location }}"
  apiKeySecret: studio-gcp-credentials
```

The declarative orchestrator references this via `modelConfig: studio-model-config`. BYO specialists continue using env vars for portability (they don't depend on Kubernetes CRDs).

### 6. Skill organization

Orchestrator routing knowledge moves to `skills/` directory:

```
skills/
├── orchestrator-routing/SKILL.md    ← routing rules, specialist descriptions
├── bundle-assembly/SKILL.md         ← OCI media types, assembly workflow
└── gemara-layers/SKILL.md           ← 7-layer model reference
```

The current `orchestrator_prompt.md` (141 lines) is split by concern. The orchestrator's `systemMessage` becomes a short identity statement; domain knowledge loads on-demand via `load_skill`.

**Why split:** The full orchestrator prompt today includes the 7-layer model reference table, bundle composition table, and template locations — all of which are only needed for specific routing decisions. Two-layer injection means they only consume tokens when relevant.

## Risks / Trade-offs

| Risk | Impact | Mitigation |
|:---|:---|:---|
| Go runtime agent-as-tool fix is 6 days old (Apr 8) | Delegation failures | Spike: deploy minimal declarative→BYO A2A round-trip before full implementation |
| 5 pods vs 1 pod | Higher resource baseline (~5x memory) | kagent Go runtime uses 7Mi idle per agent — total ~35Mi vs ~250Mi if Python |
| Gateway adds a network hop | Latency for workbench requests | Gateway and orchestrator are in-cluster — sub-millisecond hop. Streaming via EventSource unaffected. |
| Prompt drift (Git skills vs compiled) | Deployed prompts diverge from Git | Pin `skills.gitRefs.ref` to a tag or commit SHA in production. Use `main` only in dev. |
| `publish_bundle` removed from orchestrator | LLM can't auto-publish | Orchestrator returns artifacts to user; workbench UI handles publish. Add MCP wrapper later if needed. |
| kagent Declarative schema changes | Breaking CRD updates | Pin kagent Helm chart version. Track kagent releases for v1alpha2 → v1 migration. |
| Directory-level gitRefs not merged (issue #1422) | Must enumerate each skill explicitly | 3-5 skills is manageable. Revisit when the feature lands. |

## Migration Plan

1. **Spike** — Deploy a minimal declarative Agent with one `type: Agent` tool pointing at a BYO specialist. Confirm A2A round-trip on Go runtime.
2. **Gateway extraction** — Create `cmd/gateway/main.go` by extracting non-agent handlers from `cmd/agents/main.go`. Verify workbench functions identically.
3. **Specialist separation** — Add `AGENT_MODE` flag to `cmd/agents/main.go`. Create three BYO Agent CRDs.
4. **Declarative orchestrator** — Create the Agent CRD, ModelConfig CRD, and skill files. Remove `internal/agents/orchestrator.go`.
5. **Integration** — Helm upgrade, validate all pods healthy, test end-to-end delegation.
6. **Cleanup** — Remove dead code from the monolithic binary.

**Rollback:** Revert to the previous Helm chart version. The BYO orchestrator image still exists and the chart can switch `type: Declarative` back to `type: BYO`.

## Open Questions

1. **Should the `studio-agents` binary serve all three specialists or one per invocation?** Decision leans toward mode flag, but single-specialist-per-pod means independent scaling.
2. ~~**Does `ModelConfig` with `AnthropicVertexAI` provider work with GCP Workload Identity / Application Default Credentials?**~~ **Answered:** Yes for `runtime: python` — the Python Anthropic SDK handles ADC natively via `apiKeySecret`. No for `runtime: go` — the Go runtime fails with `401 CREDENTIALS_MISSING` when using `authorized_user` ADC.
3. **Should the gateway proxy A2A via HTTP reverse proxy or implement a lightweight A2A client?** HTTP reverse proxy is simpler and transport-agnostic.
4. **Agent card URL must be absolute** — the Python A2A SDK's `RemoteA2aAgent` does not resolve relative URLs. BYO agents need `A2A_BASE_URL` env var set to their Kubernetes service DNS name.
5. **Vertex AI model name format** — use `@` separator (e.g., `claude-sonnet-4@20250514`), not `-`. The dash format returns 404 from the Vertex AI Messages API.
