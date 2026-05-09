# Proposal: AgentGateway Production Routing

## User Story

As a platform operator, I need agent and tool traffic routed through AgentGateway so that A2A sessions get production-grade streaming, MCP tool calls are enforced by per-agent allowlists, and I can onboard new agents without modifying gateway code.

## Problem

Studio routes A2A traffic through a custom 200-line Go reverse proxy that forwards to the kagent controller at `:8083`. This is kagent's dev/CLI pattern (port-forward the controller, talk to agents). The production pattern kagent prescribes is AgentGateway as the A2A and MCP ingress.

Additionally, MCP tool access has no runtime enforcement. The `tools:` allowlist in `agent.yaml` is documentation — nothing blocks an agent from calling any tool the MCP server exposes.

## Scope

| In Scope | Out of Scope |
|:--|:--|
| AgentGateway standalone proxy deployment (Helm) | kagent controller deployment (already operator-managed) |
| A2A routing via HTTPRoute per agent | LLM-driven delegation (`a2a_delegate` tool) |
| MCP routing with CEL tool-access policies | MCP server image changes |
| Ingress path split (`/a2a/*` vs `/api/*`) | OAuth2 Proxy deployment (already exists) |
| Agent directory enrichment (`id`, `role`, `tools`, etc.) | Per-agent RBAC on user claims (future) |
| Removal of custom A2A proxy code | Chat history changes (independent) |
| BYO agent onboarding documentation | New agent implementations |
| Workbench endpoint update (`/a2a/`) | Workbench UI redesign |

## References

- [complytime-labs/complytime-studio#8](https://github.com/complytime-labs/complytime-studio/issues/8) — Epic: Agent Infrastructure
- [kagent A2A docs](https://kagent.dev/docs/kagent/examples/a2a-agents) — "expose the A2A endpoint publicly by using a gateway"
- [kmcp deploy docs](https://kagent.dev/docs/kmcp/deploy/server) — `kagent.dev/discovery=disabled` label for AgentGateway routing
- [AgentGateway A2A quickstart](https://agentgateway.dev/docs/kubernetes/latest/agent/a2a/)
