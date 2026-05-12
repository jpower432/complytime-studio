## Why

A consumer has a BYO agent that returns domain-specific data. The current picker pattern forces the user to switch away from the assistant to invoke it, wiping session context. The assistant has no A2A client and cannot delegate. Users must manually shuttle context between agents. This change gives the assistant the ability to call BYO agents via A2A and merge responses into its session state.

## What Changes

- Add an `a2a_delegate` tool to the assistant that invokes a registered BYO agent via AgentGateway and returns the response as structured data
- The assistant resolves the BYO agent's endpoint from the agent directory (`/api/agents`) at runtime
- BYO agent responses are stored in a `worker_responses` State field, persisted by the checkpointer
- **BREAKING**: Deprecate the agent picker dropdown as a routing mechanism. When >1 agents are registered, the picker becomes informational only (shows available capabilities) rather than switching the session target.

## Capabilities

### New Capabilities

- `a2a-delegate-tool`: LangChain tool that makes A2A calls to registered BYO agents through AgentGateway and returns structured responses into the assistant's state.

### Modified Capabilities

- `agent-picker`: The picker no longer switches the A2A routing target. It becomes an informational display of available agent capabilities. Session routing always goes to `studio-assistant`.
- `streaming-chat`: The `streamMessage` and `streamReply` functions always route to `studio-assistant` regardless of UI state. The `agentName` parameter is removed.

## Impact

- **`agents/assistant/graph.py`** — new tool added to `build_tools()`; State extended with `worker_responses`
- **`agents/assistant/tools.py`** — new `a2a_delegate` async tool function
- **`workbench/src/api/a2a.ts`** — remove `agentName` parameter; always route to `studio-assistant`
- **`workbench/src/components/chat-assistant.tsx`** — picker no longer triggers `handleNewSession()` or changes routing target
- **`workbench/src/app.tsx`** — `selectedAgent` signal repurposed or removed
- **ADR**: `docs/decisions/supervisor-session-ownership.md` (already written)
