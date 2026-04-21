## MODIFIED Requirements

### Requirement: Agent YAML supports skill references
The canonical `agent.yaml` SHALL include a `skills` array where each entry references a skill pack. Internal skills (this repo) specify `path` only. External skills specify `repo`, `ref`, and `path`. All referenced skills MUST resolve — internal paths MUST exist in the repo, external repos MUST be accessible.

#### Scenario: Internal skill reference
- **WHEN** an agent.yaml contains `skills: [{ path: "skills/gemara-mcp" }]`
- **THEN** the skill directory `skills/gemara-mcp/SKILL.md` exists in this repo

#### Scenario: External skill reference from rhaml-23/prompt
- **WHEN** an agent.yaml contains `skills: [{ repo: "https://github.com/rhaml-23/prompt.git", ref: "main", path: "skills/research.md" }]`
- **THEN** the referenced file exists in the external repo at the specified ref

#### Scenario: No dead skill references
- **WHEN** an agent.yaml is validated
- **THEN** every internal `path` resolves to a directory containing `SKILL.md`
- **THEN** every external `repo` returns 200 on HEAD request

## REMOVED Requirements

### Requirement: Agent YAML supports allowedHeaders on MCP references
**Reason**: github-mcp removed from all agents. No remaining MCP servers require per-request OBO header propagation. allowedHeaders support can be re-added if a future MCP server needs it.
**Migration**: Remove `allowedHeaders: [Authorization]` from all agent.yaml mcp entries. Already done in prior change.
