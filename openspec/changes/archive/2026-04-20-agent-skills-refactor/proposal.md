## Why

The assistant agent has zero working skills. All skill references in `agent.yaml` are dead links — `skills/gemara-layers` and `skills/gemara-authoring` point to empty directories, `gemaraproj/gemara-skills` returns 404. The 124-line `prompt.md` compensates by hardcoding SQL queries, classification tables, and domain logic inline. It never mentions the MCP servers the agent actually uses. Dead agents (`threat-modeler`, `policy-composer`) and a redundant `platform.md` add confusion. The `rhaml-23/prompt` repo has proven, loadable skills that map directly to the assistant's needs.

## What Changes

- **Delete** `agents/threat-modeler/` and `agents/policy-composer/` — no deployer exists, ConfigMap bundles unused prompts
- **Delete** `agents/platform.md` — fold useful constraints into `prompt.md`; single-agent architecture makes shared identity unnecessary
- **Create 4 internal skills** under `skills/`:
  - `gemara-mcp` — MCP tools, resources, layer model, validation workflow
  - `evidence-schema` — ClickHouse table schemas, query patterns, enum values
  - `audit-methodology` — assessment cadence rules, frequency mapping, finding classification
  - `coverage-mapping` — cross-framework join logic, strength/confidence interpretation, matrix format
- **Load 2 external skills** from `rhaml-23/prompt` via gitRef:
  - `skills/research.md` — research synthesis, source hierarchy, confidence flagging
  - `skills/gemara.md` — Gemara layer classification (supplements internal `gemara-mcp` skill)
- **Rewrite** `agents/assistant/prompt.md` to ~30 lines of workflow-only instructions
- **Update** `agents/assistant/agent.yaml` to reference real skills (remove dead links)
- **Clean** ConfigMap to only include the assistant prompt

## Capabilities

### New Capabilities

- `agent-skill-architecture`: Skill decomposition pattern — separating workflow (prompt) from domain knowledge (skills) with internal and external gitRef loading

### Modified Capabilities

- `agent-spec-skills`: Existing spec covers skill references in agent.yaml — requirements change to remove dead external repos and add real internal + `rhaml-23/prompt` gitRefs

## Impact

- `agents/` — delete 2 directories, delete `platform.md`, rewrite assistant `prompt.md` and `agent.yaml`
- `skills/` — populate with 4 SKILL.md files (currently empty)
- `charts/complytime-studio/templates/agent-prompts-configmap.yaml` — remove threat-modeler and policy-composer keys
- `charts/complytime-studio/templates/agent-specialists.yaml` — already empty, verify comment accuracy
- No API changes, no backend changes, no frontend changes
