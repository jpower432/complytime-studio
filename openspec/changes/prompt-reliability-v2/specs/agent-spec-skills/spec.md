## MODIFIED Requirements

### Requirement: Agent reads schema before authoring AuditLog
The agent prompt SHALL include a concrete AuditLog YAML template inline rather than instructing the agent to read `gemara://schema/definitions`. The template is the source of truth for field structure.

#### Scenario: Agent prepares to author AuditLog
- **WHEN** the agent reaches the "Author AuditLog" workflow step
- **THEN** the agent SHALL use the inline YAML template from the prompt to construct the artifact
- **THEN** the agent SHALL NOT need to call any MCP resource to understand AuditLog structure

#### Scenario: Agent validates authored AuditLog
- **WHEN** the agent has constructed an AuditLog YAML block
- **THEN** the agent SHALL call `validate_gemara_artifact` with `definition: "#AuditLog"` to verify correctness

## REMOVED Requirements

### Requirement: Skill packs use SKILL.md format
**Reason**: The four separate skill files (`audit-methodology`, `evidence-schema`, `coverage-mapping`, `gemara-mcp`) are consolidated into one (`studio-audit`). The SKILL.md format and frontmatter convention remain unchanged. Only the number and names of skill files change.
**Migration**: Replace four skill directories under `skills/` with `skills/studio-audit/SKILL.md`. Update any references to old skill names.
