# Design: BYO Agent Gates

## Context

The BYO gap analyst runs on KMCP infrastructure (HTTP MCP connections to
`studio-gemara-mcp:3000` and `studio-clickhouse-mcp:3000`) but currently uses
`to_a2a()`, which does not support custom `A2aAgentExecutorConfig`. Callbacks
and MCP resources are available on `LlmAgent` but are not wired into the A2A
surface, so clients never see deterministic gates, schema resources, or typed
artifact events in the stream.

## Goals

- Wire all five kagent workarounds: callbacks (before agent, after agent,
  before tool), MCP resources (where useful), and structured artifact emission
  via A2A.

## Non-goals

- Changing the KMCP infrastructure.
- Adding new MCP servers.
- Changing the agent prompt content or workflow text.

## Decisions

1. **Manual `A2aAgentExecutor` instead of `to_a2a()`**  
   Replace `to_a2a()` with a manually constructed `A2aAgentExecutor` so we can
   pass `A2aAgentExecutorConfig` (custom `adk_event_converter`, etc.). Replicate
   the roughly forty lines of setup `to_a2a()` performs, then extend with
   config. **Alternative:** fork or patch `to_a2a()` upstream — **rejected** as
   too slow for this prototype.

2. **`use_mcp_resources=True` only on gemara-mcp `McpToolset`**  
   Enable MCP resources on the Gemara toolset only. ClickHouse MCP exposes no
   useful resources for this agent; leave `use_mcp_resources` false (default)
   there.

3. **Callbacks use current ADK signatures**  
   Use `before_agent_callback(callback_context)`, `after_agent_callback(callback_context)`,
   and `before_tool_callback(tool, args, tool_context)` as `LlmAgent`
   parameters. These are independent of the A2A layer.

4. **Custom `adk_event_converter` on `A2aAgentExecutorConfig`**  
   Supply a callable that inspects ADK events for `artifact_delta`, and when
   present emits `TaskArtifactUpdateEvent` with `application/yaml` MIME metadata
   on parts.

5. **SQL guard in `before_tool`**  
   For `run_select_query`, reject queries matching a simple regex deny-list for
   DDL/DML keywords (defense in depth; not a full SQL parser).

## Risks

| Risk | Mitigation |
|:--|:--|
| ADK experimental APIs may break between releases | Pin `google-adk` to a tested version in the agent image |
| MCP resource fetch may add latency | Agent fetches schema definitions at most once per session where possible |
