## Why

The BYO gap analyst agent runs on the KMCP infrastructure but lacks the deterministic gates that motivated the BYO approach. Without callbacks, MCP resource access, and structured artifact emission, the agent is a plain LLM chat — no input validation, no SQL injection guard, no typed AuditLog artifacts in the A2A stream. These are the five kagent limitations documented in `docs/decisions/kagent-gap-catalog.md`.

## What Changes

- Wire `before_agent_callback` for input validation (policy reference + audit timeline detection)
- Wire `after_agent_callback` for output validation and `save_artifact` (AuditLog YAML extraction)
- Wire `before_tool_callback` as a SQL injection guard on ClickHouse `run_select_query`
- Enable `use_mcp_resources=True` on the gemara-mcp `McpToolset` for schema/definition reading
- Replace `to_a2a()` with manual `A2aAgentExecutor` construction to inject a custom `adk_event_converter` that emits `TaskArtifactUpdateEvent` with `application/yaml` MIME type
- Update `chat-assistant.tsx` to handle `TaskArtifactUpdateEvent` and offer "Save to Audit History"

## Capabilities

### New Capabilities

- `agent-deterministic-gates`: Before/after agent callbacks and before-tool SQL guard on the BYO gap analyst
- `agent-artifact-emission`: Custom A2A event converter emitting typed `TaskArtifactUpdateEvent` for AuditLog artifacts
- `agent-mcp-resources`: MCP resource reading for Gemara schema definitions

### Modified Capabilities

- `streaming-chat`: Chat assistant handles `TaskArtifactUpdateEvent` rendering and save-to-audit-history action
- `byo-gap-analyst`: Agent switches from `to_a2a()` to manual executor with config

## Impact

| Area | Change |
|:--|:--|
| `agents/gap-analyst/main.py` | Replace `to_a2a()` with manual `A2aAgentExecutor`, add callbacks, enable MCP resources |
| `agents/gap-analyst/callbacks.py` | Rewrite to use current ADK callback signatures |
| `agents/gap-analyst/event_converter.py` | Rewrite as `AdkEventToA2AEventsConverter` callable |
| `agents/gap-analyst/Dockerfile` | Add `callbacks.py` and `event_converter.py` back to COPY |
| `workbench/src/components/chat-assistant.tsx` | Handle `TaskArtifactUpdateEvent`, save-to-audit action |
| `workbench/src/api/a2a.ts` | Ensure `onArtifact` callback fires for artifact events |
