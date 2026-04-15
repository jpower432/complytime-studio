## 1. Spike — Validate Declarative→BYO A2A Round-Trip

- [x] 1.1 Deploy a minimal Declarative Agent CRD (`runtime: go`) with a `type: Agent` tool referencing an existing BYO specialist
- [x] 1.2 Send a test message to the declarative agent and confirm it delegates to the BYO specialist via A2A
- [x] 1.3 Document any issues (latency, error handling, agent card discovery) and clean up test resources

### Spike Findings

**Result: Validated.** A kagent Declarative agent successfully delegates to a BYO specialist via A2A.

**Issues discovered and resolved:**

| # | Issue | Resolution |
|:--|:------|:-----------|
| 1 | Go runtime `401 CREDENTIALS_MISSING` for AnthropicVertexAI with `authorized_user` ADC | Switched to `runtime: python` — Python SDK handles ADC natively |
| 2 | Model name format `claude-sonnet-4-20250514` returns 404 on Vertex AI Messages API | Use `claude-sonnet-4@20250514` (`@` separator required by Vertex) |
| 3 | BYO agent card returns `null` for `defaultInputModes`/`defaultOutputModes` | Added `[]string{"text"}` for both fields in all agent cards |
| 4 | BYO agent card URL `/invoke` (relative) — Python A2A SDK cannot resolve | Added `A2A_BASE_URL` env var; agent card now returns absolute service DNS URL |

**Design decisions for remaining phases:**

- **Declarative orchestrator MUST use `runtime: python`** until kagent Go runtime adds `authorized_user` ADC support for AnthropicVertexAI
- **All agent cards MUST include** `DefaultInputModes` and `DefaultOutputModes` as non-null lists
- **BYO agents need `A2A_BASE_URL`** env var set to their Kubernetes service DNS to serve correct agent cards
- **Latency**: Round-trip through declarative→BYO is ~10-13s for a simple query (includes 2 LLM calls)
- Task 4.3 updated: use `runtime: python` instead of `runtime: go`

## 2. Specialist Agent Separation

- [x] 2.1 Add `AGENT_MODE` env var support to `cmd/agents/main.go` — `threat-modeler`, `gap-analyst`, `policy-composer`, or `all` (default)
- [x] 2.2 In single-mode, start only the selected specialist agent and its A2A server on port 8080
- [x] 2.3 Ensure `/.well-known/agent.json` agent card is served for each specialist mode
- [x] 2.4 Create three BYO Agent CRD templates in the Helm chart (`studio-threat-modeler`, `studio-gap-analyst`, `studio-policy-composer`)
- [x] 2.5 Each BYO CRD sets `AGENT_MODE`, `MCP_TRANSPORT=sse`, and model provider env vars
- [x] 2.6 Verify each specialist pod starts independently and serves its A2A agent card

## 3. Gateway Extraction

- [x] 3.1 Create `cmd/gateway/main.go` extracting: workbench SPA, `/api/validate`, `/api/migrate`, `/api/registry/*`, `/api/publish`
- [x] 3.2 Add A2A reverse proxy handler — forward `/invoke` and `/.well-known/*` to orchestrator service URL (configurable via `ORCHESTRATOR_URL` env var)
- [x] 3.3 Build `studio-gateway` Docker image (Dockerfile.gateway)
- [x] 3.4 Create Helm chart templates for gateway Deployment and Service
- [x] 3.5 Verify workbench loads, validate/migrate proxy works, registry browser works, publish works through gateway

## 4. Declarative Orchestrator

- [x] 4.1 Create `ModelConfig` CRD template in Helm chart (`studio-model-config`) with AnthropicVertexAI provider config from values
- [x] 4.2 Create skill files: `skills/orchestrator-routing/SKILL.md`, `skills/bundle-assembly/SKILL.md`, `skills/gemara-layers/SKILL.md` — extracted from `orchestrator_prompt.md`
- [x] 4.3 Create Declarative Agent CRD template (`studio-orchestrator`) with `runtime: python`, `modelConfig` reference, `skills.gitRefs`, and `tools` array
- [x] 4.4 Tools array: `type: Agent` for each specialist + `type: McpServer` for `studio-oras-mcp` with tool name filter
- [x] 4.5 Set `systemMessage` to a short orchestrator identity prompt (routing knowledge loads via skills)
- [x] 4.6 Configure GCP credentials volume mount on the declarative agent's deployment spec

## 5. Integration and Validation

- [x] 5.1 Update `values.yaml` with new configuration sections (gateway, specialist modes, model config)
- [x] 5.2 `helm upgrade` and verify all pods reach Ready state (gateway, orchestrator, 3 specialists, 3 MCP servers)
- [x] 5.3 Send a threat modeling request through the gateway → orchestrator → threat modeler and confirm end-to-end delegation
- [x] 5.4 Verify workbench SPA loads via gateway and can validate/publish artifacts
- [ ] 5.5 Verify `load_skill` appears in orchestrator's available tools (deferred — requires git repo skills config)

## 6. Cleanup

- [x] 6.1 Remove orchestrator-specific code from `cmd/agents/main.go` (registerOrchestrator, registerProxy, registerRegistryProxy, registerPublishEndpoint, registerWorkbench)
- [x] 6.2 Remove `internal/agents/orchestrator.go` and `orchestrator_prompt.md` (replaced by CRD + skills)
- [x] 6.3 Remove old `agent-orchestrator.yaml` BYO template from Helm chart
- [x] 6.4 Update `Makefile` targets for new image builds (gateway, agents)
- [x] 6.5 Update chart `NOTES.txt` with new architecture description
