## MODIFIED Requirements

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

#### Scenario: MCP tools reference PostgreSQL and OpenSearch backends
- **WHEN** the assistant `agent.yaml` declares MCP tools
- **THEN** `studio-clickhouse-mcp` SHALL be replaced with `studio-postgres-mcp` (for knowledge graph queries) and `studio-evidence-mcp` (for evidence search routed through the bulk store adapter)
