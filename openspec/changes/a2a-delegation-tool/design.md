## Context

The assistant (`agents/assistant/graph.py`) has MCP tool access (gemara-mcp, postgres-mcp) but no A2A client capability. A consumer has a BYO agent registered in `agentDirectory` that returns domain-specific data. The workbench currently routes A2A calls based on a `selectedAgent` signal -- switching agents wipes the session. The assistant needs to call BYO agents as part of its workflow without the user leaving the conversation.

AgentGateway already routes A2A traffic via HTTPRoutes (`/a2a/{agent-id}`). The assistant pod can reach AgentGateway at `http://agentgateway-proxy`. CEL policies control which agent can call which tools. The transport layer exists -- what's missing is the assistant-side client.

## Goals / Non-Goals

**Goals:**
- Assistant can invoke a registered BYO agent and receive structured data back
- Session context is preserved across delegation (no wipe)
- BYO agent responses are stored in State and available to subsequent LLM turns
- Workbench always routes to `studio-assistant` regardless of picker state

**Non-Goals:**
- Dynamic agent discovery (ANS) -- use existing `agentDirectory` / `GET /api/agents`
- Verification gate on worker responses -- BYO agent returns domain data, not Gemara artifacts
- Context compaction before dispatch -- defer until payload size is a measured problem
- Multi-turn delegation (streaming conversation with BYO agent) -- single request/response only
- Agent-to-agent authentication beyond existing `X-Agent-ID` header

## Decisions

### D1: Delegation as a graph node, not an LLM tool

**Decision:** Implement delegation as a dedicated `delegate` graph node triggered by a conditional edge. The routing decision (does this workflow need BYO data?) is deterministic -- based on policy metadata or an explicit state flag set during evidence assembly.

**Rationale:** In compliance, predictability matters more than flexibility. A graph node is:
- Auditable: visible as a distinct step in LangGraph Studio traces
- Testable: plain Python function, no LLM in the routing decision
- Reliable: fires when the condition is met, never forgotten or hallucinated
- Zero token cost: no tool description consuming context on every turn
- Extensible: adding a second BYO agent is a routing table change, not a prompt rewrite

The trigger condition is deterministic: when the policy or evidence type requires domain-specific data that only the BYO agent can provide, the graph routes through the `delegate` node. This is knowable from policy metadata at evidence assembly time.

**Alternative rejected:** LLM tool call. Makes the routing decision probabilistic. The LLM can forget to delegate or delegate unnecessarily. In compliance, "the system didn't ask for required data" is an audit finding -- unacceptable as a stochastic outcome.

### D2: Resolve agent endpoint from gateway at call time

**Decision:** The tool calls `GET /api/agents` (internal gateway) to resolve the BYO agent's URL at invocation time. No hardcoded endpoints.

**Rationale:** The `agentDirectory` already contains agent URLs. Resolving at call time means re-deploys, scaling events, or new agent registrations are reflected without restarting the assistant. Uses existing infrastructure.

**Alternative rejected:** Static env var per BYO agent. Doesn't scale to N agents without restart. Couples assistant deployment to agent deployment.

### D3: A2A call via httpx, not MCP

**Decision:** The delegation tool makes a direct HTTP POST to the BYO agent's A2A endpoint (JSON-RPC `message/send`) using `httpx`. Response is parsed and returned as tool output.

**Rationale:** A2A is JSON-RPC over HTTP, not MCP. The `langchain-mcp-adapters` client doesn't apply here. A simple httpx call with the A2A message envelope is sufficient. The response is a single `Task` object with `artifacts`.

**Alternative rejected:** Wrapping the BYO agent as an MCP server. Would require protocol adaptation on the BYO side. A2A is the native protocol -- use it directly.

### D4: Store worker responses in a dedicated State field

**Decision:** The `delegate` node writes the BYO agent's response into a `worker_data: dict` State field keyed by agent ID. The LLM receives the data as injected context on its next turn.

**Rationale:** As a graph node (not a tool), the response doesn't automatically appear as a `ToolMessage`. A dedicated State field:
- Survives message window truncation (top-level state, not buried in messages)
- Is available to all downstream nodes (the verification gate can reference it)
- Provides clear provenance: "this data came from agent X at time T"
- Enables the LLM to reference it without re-fetching

**Alternative rejected:** Injecting as a synthetic `HumanMessage`. Pollutes the message history with data the user didn't send. Confuses conversation replay.

### D5: Workbench migration -- remove agentName routing

**Decision:** Remove the `agentName` parameter from `streamMessage` and `streamReply`. Always route to `studio-assistant`. The picker dropdown becomes a read-only capability display (or is removed entirely).

**Rationale:** With delegation handled by the assistant, the user never needs to switch targets. The picker created a false choice. Removing it simplifies the A2A client and eliminates the session-wipe bug.

**Migration:** If `availableAgents.length > 1`, the picker can display agent names/descriptions as "Available capabilities" without being a routing control. Or remove it entirely. UX decision, not architectural.

## Risks / Trade-offs

| Risk | Mitigation |
|:--|:--|
| BYO agent is unreachable | Tool returns error message; LLM reports to user. Standard tool error handling. |
| BYO agent returns massive payload | Set httpx timeout (30s) and response size cap (1MB). Truncate with summary if exceeded. |
| LLM calls delegate tool unnecessarily | Tool description must be specific about when to use it. Include the BYO agent's skill description in the tool docstring. |
| AgentGateway CEL policy blocks assistant→BYO call | Add a policy rule allowing `X-Agent-ID: studio-assistant` to call the BYO agent's A2A endpoint. This is a route, not an MCP tool -- may need a separate policy type. |
| Picker removal breaks existing user workflows | No users rely on the picker for routing today (only one real agent). Migration risk is zero. |

## Open Questions

1. Does AgentGateway's CEL `AuthorizationPolicy` apply to A2A routes, or only MCP tool calls? If A2A-only, no policy change needed. Need to verify.
2. Should the tool description include the BYO agent's skill tags dynamically (fetched from `/api/agents`), or be static? Dynamic is more accurate but adds a fetch to tool construction.
3. What is the A2A response format the BYO agent returns? Need to confirm it follows standard A2A `Task` envelope with `artifacts[].parts[].text`.
