# Tasks: LangGraph Agent Runtime

## Container image
- [ ] Create `agents/langgraph/` directory with `main.py`, `Dockerfile`, `requirements.txt`
- [ ] Implement A2A entrypoint using `kagent-langgraph` `KAgentApp`
- [ ] Implement `SpecLoader` (constitution, persona, command, function loading from markdown)
- [ ] Implement `build_agent` / `build_llm` with LLM provider selection (`google`, `anthropic`)
- [ ] Implement tool registry (`init_tools`, `get_all_tools`, `get_tools_for_command`, `COMMAND_TOOL_MAP`)
- [ ] Wire `AsyncPostgresSaver` checkpointer for multi-turn chat
- [ ] Wire `validate_command_output` MCP tool call in post-processing (per governance spec)
- [ ] Write agent persona specs: `program-agent.md`, `evidence-agent.md`, `coordinator.md`
- [ ] Write command specs for v1 commands (markdown with YAML frontmatter)
- [ ] Build and test image locally with `AGENT_TYPE=program`

## Helm / kagent
- [ ] Add `agents/langgraph/agent.yaml` canonical spec (per AGENTS.md pattern)
- [ ] Add BYO Agent CRD Helm template parameterized by `AGENT_TYPE`
- [ ] Add per-persona toggles in `values.yaml` (`agents.programAgent.enabled`, etc.)
- [ ] Add agent entries to `agentDirectory` in `values.yaml` for each enabled persona
- [ ] Wire `GEMARA_MCP_URL`, `CLICKHOUSE_MCP_URL`, `KNOWLEDGE_BASE_MCP_URL`, `POSTGRES_URL` env vars

## Integration
- [ ] Verify A2A routing: gateway → kagent controller → LangGraph agent pod
- [ ] Verify command execution: workbench → gateway → A2A → agent → command spec → SSE response
- [ ] Verify multi-turn chat: checkpointer persists state across turns
- [ ] Verify quality gate MCP tool call returns results in A2A response metadata
- [ ] Verify MCP tool calls: agent → gemara-mcp validate, agent → clickhouse-mcp query
- [ ] Test with `AGENT_TYPE` values: `program`, `evidence`, `coordinator`
