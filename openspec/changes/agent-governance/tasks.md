# Tasks: Agent Governance

## Constitution
- [ ] Create `skills/constitution/SKILL.md` with invariant behavioral rules (under 500 tokens)
- [ ] Add constitution to `agents/assistant/agent.yaml` skills block
- [ ] Add constitution to LangGraph agent skill loading (spec loader)
- [ ] Verify prompt assembly order: platform.md → constitution → persona → domain skills

## Quality Gates MCP Tool
- [ ] Add `validate_command_output` tool to gemara-mcp server
- [ ] Implement structural check: expected sections as `##` headings
- [ ] Implement format checks: no numbered headings, no emoji, valid JSON blocks
- [ ] Return structured result (structural + format + overall)
- [ ] Wire quality gate call into assistant's `after_agent` callback
- [ ] Wire quality gate call into LangGraph agent post-processing
- [ ] Add quality gate badge rendering to workbench command output panel
- [ ] Tests: structural pass/fail, format pass/fail, combined results
