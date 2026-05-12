## 1. Delegate Node (Python)

- [ ] 1.1 Create `agents/assistant/delegate.py` with `delegate_node(state: AuditState)` async function
- [ ] 1.2 Implement agent directory resolution: `GET {GATEWAY_URL}/api/agents` → find by target agent ID → extract URL
- [ ] 1.3 Implement A2A `message/send` call via httpx: 30s timeout, 1MB cap, `X-Agent-ID: studio-assistant` header
- [ ] 1.4 Parse A2A Task response: extract `artifacts[].parts[].text` or error from `status.state`
- [ ] 1.5 Write response (or error dict) to `state.worker_data` keyed by agent ID
- [ ] 1.6 Write unit tests: success, agent not found, timeout, oversized response, failed status

## 2. State Extension

- [ ] 2.1 Add `worker_data: dict` field to State (or `AuditState` if verification harness lands first)
- [ ] 2.2 Add `needs_delegation: bool` flag to State (defaults to `false`)
- [ ] 2.3 Add `delegation_target: str` field to State (agent ID to delegate to, empty string default)

## 3. Graph Wiring

- [ ] 3.1 Add `delegate` node to audit subgraph (or main graph if verification harness hasn't landed)
- [ ] 3.2 Add conditional edge: after evidence assembly, route to `delegate` if `needs_delegation == true`
- [ ] 3.3 Add edge: `delegate` → agent node (LLM receives worker data on next turn)
- [ ] 3.4 Add skip edge: if `needs_delegation == false`, bypass `delegate` and continue to classification
- [ ] 3.5 Define trigger condition: policy metadata check or `needs_delegation` flag set by LLM during assembly

## 4. Context Injection

- [ ] 4.1 Modify `agent_node` to inject `state.worker_data` into system messages when populated
- [ ] 4.2 Format injected worker data with provenance header: `--- Data from <agent-id> (reference only) ---`

## 5. Prompt Update

- [ ] 5.1 Add guidance to `prompt.md`: "When worker data is present in context, incorporate it into your analysis. Do not re-request data that has already been provided by a worker agent."
- [ ] 5.2 Document the delegation trigger: which policies/conditions cause delegation
- [ ] 5.3 Run `make sync-prompts`

## 6. Workbench — Remove Agent Routing

- [ ] 6.1 Remove `agentName` parameter from `streamMessage()` in `workbench/src/api/a2a.ts`
- [ ] 6.2 Remove `agentName` parameter from `streamReply()` in `workbench/src/api/a2a.ts`
- [ ] 6.3 Hardcode `a2aEndpoint()` to return `/a2a/studio-assistant`
- [ ] 6.4 Remove `handleNewSession()` call from picker `onChange` in `chat-assistant.tsx`
- [ ] 6.5 Convert picker from routing control to informational display (or remove entirely)
- [ ] 6.6 Remove `selectedAgent` signal usage from routing call sites

## 7. AgentGateway Policy

- [ ] 7.1 Verify whether CEL `AuthorizationPolicy` applies to A2A routes or MCP-only
- [ ] 7.2 If needed: add policy rule allowing `studio-assistant` to reach BYO agent A2A endpoints
- [ ] 7.3 Verify HTTPRoute exists for BYO agent (it should from `agentgateway-routes.yaml` if agent is in `agentDirectory`)

## 8. Integration Testing

- [ ] 8.1 Test: graph routes through delegate node when policy triggers delegation
- [ ] 8.2 Test: graph skips delegate node when policy does not require BYO data
- [ ] 8.3 Test: LLM sets `needs_delegation` flag, graph routes to delegate on next cycle
- [ ] 8.4 Test: BYO agent down → error in `worker_data` → LLM reports gracefully
- [ ] 8.5 Test: worker_data persists in State after checkpointer resume
- [ ] 8.6 Test: workbench always routes to `studio-assistant` regardless of UI state
