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

#### Scenario: Posture-check skill registered
- **WHEN** the assistant `agent.yaml` includes `skills: [{ path: "skills/posture-check" }]`
- **THEN** Helm renders a kagent `gitRefs` entry for `skills/posture-check` and the agent can load the posture-check skill at runtime

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

### Requirement: Agent reads schema before authoring AuditLog
The agent prompt SHALL instruct the assistant to read the `gemara://schema/definitions` MCP resource to obtain the `#AuditLog` definition BEFORE authoring any AuditLog artifact. This ensures the agent uses the correct field structure.

#### Scenario: Agent prepares to author AuditLog
- **WHEN** the agent reaches the "Author AuditLog" workflow step
- **THEN** the agent SHALL first read `gemara://schema/definitions` to obtain the `#AuditLog` schema definition

### Requirement: Agent validates AuditLog via MCP tool
The agent prompt SHALL instruct the assistant to call the `validate_gemara_artifact` MCP tool with `definition: "#AuditLog"` on every generated AuditLog YAML block before publishing it. The agent SHALL fix validation errors and re-validate up to 3 times.

#### Scenario: Agent produces valid AuditLog
- **WHEN** the agent generates an AuditLog YAML artifact
- **THEN** the agent SHALL call `validate_gemara_artifact` before calling `publish_audit_log`

#### Scenario: Validation fails on first attempt
- **WHEN** `validate_gemara_artifact` returns errors
- **THEN** the agent SHALL fix the identified issues and re-validate, up to 3 total attempts

#### Scenario: Validation fails after 3 attempts
- **WHEN** the agent cannot produce valid YAML after 3 validation attempts
- **THEN** the agent SHALL report the validation errors to the user and halt

### Requirement: Agent recognizes sticky-notes context tag

The agent prompt SHALL document the `<sticky-notes>` tag convention. Content within `<sticky-notes>` tags represents persistent user-curated facts. The agent SHALL treat these as always-true background context unless explicitly contradicted by the user.

#### Scenario: Sticky notes present in message
- **WHEN** a user message contains a `<sticky-notes>` block
- **THEN** the agent SHALL treat each note as a persistent fact for the duration of the conversation
- **THEN** the agent SHALL NOT ask the user to re-confirm information already in sticky notes

#### Scenario: User contradicts a sticky note
- **WHEN** the user provides information that contradicts a sticky note
- **THEN** the agent SHALL use the user's latest statement and note the discrepancy

### Requirement: Agent suggests sticky notes for persistent facts

The agent prompt SHALL instruct the assistant to suggest saving persistent facts as sticky notes when the user establishes scope, dates, priorities, or recurring parameters.

#### Scenario: User establishes audit window
- **WHEN** the user states "our audit window is Q1 2026" or equivalent scope-setting fact
- **THEN** the agent SHALL include a suggestion: "Tip: save 'Audit window: Q1 2026' as a sticky note to carry this across sessions."

#### Scenario: Agent does not auto-create
- **WHEN** the agent suggests a sticky note
- **THEN** the agent SHALL NOT create the note automatically
- **THEN** the user SHALL manually add the note via the sticky notes panel
