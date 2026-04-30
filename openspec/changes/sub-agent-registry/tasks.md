# Tasks: Sub-Agent Registry

- [ ] Add `ID` and `Status` fields to `agents.Card` struct
- [ ] Add `Role`, `Framework`, `Delegatable`, `Examples`, `Tools` fields to `agents.Card`
- [ ] Validate `id` uniqueness at startup, reject duplicates
- [ ] Update `GET /api/agents` to return enriched cards, exclude `status: hidden`
- [ ] Expand `agentDirectory` in `values.yaml` with new fields and example entries
- [ ] Update Helm gateway template to render expanded `AGENT_DIRECTORY` JSON
- [ ] Generate assistant sub-agent context block in Helm prompt ConfigMap from delegatable entries
- [ ] Add `a2a_delegate` tool to Studio assistant (ADK tool calling gateway A2A proxy)
- [ ] Implement delegation guardrails: max depth 2, no self-delegation, 120s timeout
- [ ] Update workbench agent picker to display role badge, framework badge, examples, skill tags
- [ ] Update workbench chat routing to support selecting a specific agent by `id`
- [ ] Add kagent BYO agent Helm templates for bundled sub-agents
- [ ] Document extension agent pattern for proprietary integrations
- [ ] Test: assistant delegates to sub-agent, response streams back to workbench
- [ ] Test: docker-compose with `AGENT_DIRECTORY` env var (backward compat)
- [ ] Test: duplicate `id` in directory rejected at startup
