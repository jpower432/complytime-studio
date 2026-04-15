## ADDED Requirements

### Requirement: Agent YAML supports skill references
The canonical `agent.yaml` SHALL include a `skills` array where each entry references a git-based skill pack. Internal skills (this repo) specify `path` only. External skills specify `repo`, `ref`, and `path`.

#### Scenario: Internal skill reference
- **WHEN** an agent.yaml contains `skills: [{ path: "skills/gemara-layers" }]`
- **THEN** Helm renders a kagent `gitRefs` entry with the platform repo URL, `ref: main`, and `path: skills/gemara-layers`

#### Scenario: External skill reference
- **WHEN** an agent.yaml contains `skills: [{ repo: "https://github.com/org/skills.git", ref: "main", path: "skills/stride-analysis" }]`
- **THEN** Helm renders a kagent `gitRefs` entry with the external URL, ref, and path passed through directly

#### Scenario: No skills defined
- **WHEN** an agent.yaml omits the `skills` field
- **THEN** Helm renders the Agent CRD without a `spec.skills` block and the agent operates with prompt-only knowledge

### Requirement: Skill packs use SKILL.md format
Each skill pack directory SHALL contain a `SKILL.md` file with YAML frontmatter (`name`, `description`) followed by markdown instructions. kagent's init container loads these into `/skills` at runtime.

#### Scenario: Agent discovers skill at runtime
- **WHEN** an agent pod starts with a gitRefs skill reference
- **THEN** kagent's init container clones the repo and mounts the skill directory under `/skills/<name>/`
- **THEN** the agent can read `SKILL.md` via the skill tool to load domain knowledge on demand

### Requirement: Agent YAML supports allowedHeaders on MCP references
Each `mcp[]` entry in agent.yaml SHALL support an `allowedHeaders` string array specifying which A2A request headers to propagate to that MCP server's tool calls.

#### Scenario: OBO header declaration
- **WHEN** an agent.yaml MCP entry includes `allowedHeaders: [Authorization]`
- **THEN** Helm renders `McpServerTool.allowedHeaders: ["Authorization"]` on that tool reference in the Agent CRD

#### Scenario: No allowedHeaders
- **WHEN** an agent.yaml MCP entry omits `allowedHeaders`
- **THEN** Helm renders the tool reference without `allowedHeaders` and no A2A request headers are forwarded
