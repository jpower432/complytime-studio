# Tasks: AgentGateway Production Routing

## Phase 1 — ADR + AgentGateway Deployment

| ID | Task | Status |
|:---|:--|:--:|
| P1-1 | Write ADR `docs/decisions/agentgateway-production-routing.md` | [ ] |
| P1-2 | Add AgentGateway install to `deploy/kind/setup.sh` (Gateway API CRDs + proxy) | [ ] |
| P1-3 | Add `agentgateway` values block to `charts/complytime-studio/values.yaml` | [ ] |
| P1-4 | Create Helm template: `Gateway` resource (`agentgateway-proxy`) | [ ] |
| P1-5 | Create Helm template: AgentGateway Deployment + Service | [ ] |
| P1-6 | Configure ingress path split: `/a2a/*` → AgentGateway, else → Studio Gateway | [ ] |

## Phase 2 — A2A Route Configuration

| ID | Task | Status |
|:---|:--|:--:|
| P2-1 | Create Helm template: `AgentgatewayBackend` (ranged over `agentDirectory`) | [ ] |
| P2-2 | Create Helm template: `HTTPRoute` per agent (path prefix `/{id}`) | [ ] |
| P2-3 | Verify A2A streaming works through AgentGateway (SSE pass-through) | [ ] |

## Phase 3 — MCP Routing + Tool Enforcement

| ID | Task | Status |
|:---|:--|:--:|
| P3-1 | Add `kagent.dev/discovery=disabled` label to MCPServer CRD templates | [ ] |
| P3-2 | Create Helm template: MCP `AgentgatewayBackend` (gemara, oras, postgres targets) | [ ] |
| P3-3 | Create Helm template: MCP `HTTPRoute` | [ ] |
| P3-4 | Create Helm template: `AuthorizationPolicy` with CEL rules from `agentDirectory[].tools` | [ ] |
| P3-5 | Update `byo-assistant.yaml` env vars: MCP URLs point to AgentGateway | [ ] |
| P3-6 | Configure agent pods to send `X-Agent-ID` header on MCP requests | [ ] |

## Phase 4 — Gateway Cleanup + Workbench

| ID | Task | Status |
|:---|:--|:--:|
| P4-1 | Delete `registerA2AProxy`, `RegisterA2AProxy`, `RegisterA2AForward` from `internal/agents/agents.go` | [ ] |
| P4-2 | Delete `Options.KagentA2AURL`, `Options.AgentNamespace` fields | [ ] |
| P4-3 | Remove `KAGENT_A2A_URL`, `KAGENT_AGENT_NAMESPACE`, `A2A_PROXY_URL` from `cmd/gateway/main.go` | [ ] |
| P4-4 | Remove `KAGENT_A2A_URL`, `KAGENT_AGENT_NAMESPACE` from gateway Helm template | [ ] |
| P4-5 | Update `workbench/src/api/a2a.ts`: `a2aEndpoint()` returns `/a2a/${agentName}` | [ ] |
| P4-6 | Remove `/api/a2a/` exemption from `writeProtect` middleware (dead path) | [ ] |

## Phase 5 — Agent Directory Enrichment

| ID | Task | Status |
|:---|:--|:--:|
| P5-1 | Extend `Card` struct: add `ID`, `Role`, `Framework`, `Status`, `Tools`, `Examples`; remove `URL` | [ ] |
| P5-2 | Update `registerDirectory`: filter `status: hidden` before serializing | [ ] |
| P5-3 | Update `agentDirectory` in values.yaml with enriched schema for `studio-assistant` | [ ] |
| P5-4 | Update gateway Helm template `AGENT_DIRECTORY` env construction for new fields | [ ] |

## Phase 6 — Documentation

| ID | Task | Status |
|:---|:--|:--:|
| P6-1 | Write BYO agent onboarding guide (3-step: container, CRD, directory entry) | [ ] |
| P6-2 | Update `AGENTS.md` to reflect AgentGateway routing (remove kagent controller A2A references) | [ ] |
| P6-3 | Update `deploy/kind/setup.sh` README section | [ ] |
