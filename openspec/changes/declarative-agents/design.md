## Context

Three specialist agents (threat modeler, gap analyst, policy composer) are deployed as a BYO binary (`cmd/agents`) with ~600 lines of Go wiring code. Each agent follows an identical pattern: embed a prompt, wire MCP toolsets with filters, call `llmagent.New`. The orchestrator adds an LLM routing layer that duplicates what the platform UI can do deterministically. kagent v1alpha2 Declarative Agent CRDs now support `runtime: go`, `toolNames` filtering on `RemoteMCPServer` references, `a2aConfig` for built-in A2A servers, and `systemMessageFrom` for ConfigMap-backed prompts.

The gateway already serves `/api/publish` and `/api/registry/*` REST endpoints. It does not currently expose an agent directory.

## Goals / Non-Goals

**Goals:**

- Replace all BYO Agent CRDs with Declarative Agent CRDs — zero Go agent code
- Establish canonical agent definitions (`agents/<name>/agent.yaml` + `prompt.md`) as the framework-independent source of truth
- Delete `cmd/agents/`, `internal/agents/`, the agents Docker image, and the orchestrator
- Expose specialist agent cards via a gateway API endpoint for frontend routing
- Keep the publish bundle workflow in the gateway (already there at `/api/publish`)

**Non-Goals:**

- Multi-agent chaining or workflow orchestration (users chain manually via UI)
- Custom agent logic in Go (if needed later, revisit with BYO or tRPC-Agent-Go)
- Local dev without Kubernetes (kind/minikube is acceptable; a non-k8s local runner is a separate change)
- Migrating to a non-kagent runtime (the canonical agent format hedges against this, but execution is deferred)

## Decisions

### 1. Canonical agent definition format

Each agent lives in `agents/<name>/` with two files:

- `agent.yaml` — identity, MCP tools, A2A skills, model reference
- `prompt.md` — system prompt as plain markdown

This format is framework-agnostic. The Helm chart renders it into kagent CRDs. If we pivot runtimes, we write a different renderer.

**Alternative considered:** Write kagent CRDs directly in templates. Rejected — couples agent identity to one runtime's schema. The canonical format costs one Helm template and buys full portability.

### 2. Declarative Agent CRDs with Go runtime

Use `runtime: go` in `DeclarativeAgentSpec`. kagent manages the agent pod, A2A server, session, and model wiring. No custom binary.

**Alternative considered:** `runtime: python`. Rejected — Go runtime has faster startup and the team works in Go. Python runtime offers no advantage since we have no Python agent logic.

### 3. Prompts in ConfigMap

Agent prompts rendered into a single ConfigMap (`studio-agent-prompts`) from the `agents/<name>/prompt.md` files. Each agent's `systemMessageFrom` references its key.

**Alternative considered:** Inline `systemMessage` in the Agent CRD. Rejected — prompts are long markdown documents that would bloat the CRD YAML and make Helm values unwieldy.

### 4. ModelConfig CRD shared across agents

One `ModelConfig` CRD (`studio-model`) referenced by all Declarative agents. Provider and model name configured via `values.yaml`.

**Alternative considered:** Per-agent ModelConfig. Rejected — all agents use the same model today. Per-agent configs can be added later without structural changes.

### 5. Delete orchestrator entirely

No LLM routing layer. The gateway exposes an `/api/agents` endpoint that returns agent cards (name, description, skills, A2A URL). The frontend presents a specialist directory. Users pick directly.

**Alternative considered:** Keep orchestrator as an optional Declarative agent for ambiguous multi-step requests. Deferred — can be re-added as a Declarative agent later if user research shows demand. No code investment now.

### 6. Publish stays in gateway

`/api/publish` already exists in the gateway. The `publish_bundle` function tool in `internal/publish/tool.go` is deleted since no agent calls it. `internal/publish/bundle.go` and its dependencies remain, used by the gateway handler.

**Alternative considered:** Make publish an MCP server so agents could call it. Rejected — the user decided publish is a platform/frontend concern, not an agent action.

## Risks / Trade-offs

| Risk | Mitigation |
|:--|:--|
| kagent Go runtime missing a feature we need later (e.g., custom callbacks) | Canonical agent format allows pivot to BYO or tRPC-Agent-Go; Go runtime covers prompt + tools today |
| Local dev requires kind/minikube + kagent operator | Acceptable for now; a non-k8s local runner is a separable future change |
| Prompt changes require ConfigMap update + pod restart | kagent watches ConfigMap changes and rolls pods; standard k8s pattern |
| Loss of orchestrator for multi-step missions | UI-driven chaining (user passes output from one specialist to another); kagent agent-as-tool for future automation |
| `externalize-agent-skills` change becomes stale | Cancel it — this change supersedes its scope |

## Migration Plan

1. Create `agents/` canonical directory with definitions for all three specialists
2. Add Helm templates: `model-config.yaml`, `agent-prompts-configmap.yaml`, rewrite `agent-specialists.yaml`
3. Delete `cmd/agents/`, `internal/agents/`, agents Dockerfile
4. Delete `agents/orchestrator.md`, `skills/orchestrator-routing/`
5. Delete `internal/publish/tool.go` (function tool); keep `bundle.go`, `media_types.go`, `helpers.go`, `sign.go`
6. Add `/api/agents` endpoint to gateway
7. Update `values.yaml` — remove `agents.image`, add `model` config section
8. Clean up `go.mod` — remove unused agent dependencies

## Open Questions

- **kagent version pinning**: Which kagent release introduced `runtime: go`? Pin Helm chart dependency to that minimum.
- **Agent-as-tool**: Should policy composer be able to invoke threat modeler via kagent's cross-agent tool mechanism, or is UI chaining sufficient for now?
