## ADDED Requirements

### Requirement: System prompt total size under 15K chars
The assembled system prompt (prompt.md + skills + preloaded resources + few-shot examples) SHALL NOT exceed 15,000 characters.

#### Scenario: Prompt assembly at startup
- **WHEN** the agent starts and `_build_instruction()` assembles the full system prompt
- **THEN** the total character count SHALL be under 15,000

### Requirement: No SQL templates in system prompt
The system prompt SHALL NOT contain SQL query patterns with placeholder syntax. The agent SHALL construct queries from table metadata and column names.

#### Scenario: Agent queries ClickHouse
- **WHEN** the agent needs to query a ClickHouse table
- **THEN** the agent SHALL construct the SQL query using known table and column names
- **THEN** the query SHALL contain only literal string values, never template placeholders

#### Scenario: Agent discovers table structure
- **WHEN** the agent is unsure of a table's columns or types
- **THEN** the agent SHALL run `DESCRIBE TABLE <table_name>` via the `run_select_query` tool

### Requirement: Compact table reference in skill
The consolidated skill SHALL list all ClickHouse tables with column names in a compact one-line-per-table format. No type annotations, no descriptions, no example queries.

#### Scenario: Table reference format
- **WHEN** the skill lists a table
- **THEN** the format SHALL be `table_name: col1, col2, col3`
- **THEN** all tables SHALL fit within 1,500 characters total

### Requirement: Inline AuditLog template in prompt
The `prompt.md` workflow step for authoring AuditLogs SHALL contain a concrete YAML template showing the exact field structure, with explicit annotations for fields the LLM commonly confuses.

#### Scenario: Template includes critical field rules
- **WHEN** the prompt defines the AuditLog template
- **THEN** the template SHALL include a comment on `criteria-reference.entries[].reference-id` stating it MUST be `reference-id`, not `entry-id`
- **THEN** the template SHALL include `metadata.mapping-references` as a required block

#### Scenario: Template is self-contained
- **WHEN** the agent reaches the "Author AuditLog" workflow step
- **THEN** the template SHALL be immediately adjacent to that step in the prompt
- **THEN** the agent SHALL NOT need to load additional resources to understand the AuditLog structure

### Requirement: MappingDocuments auto-queried from ClickHouse
The prompt SHALL instruct the agent to query `mapping_documents` by `policy_id` as a workflow step. MappingDocuments SHALL NOT be listed as a user-provided input.

#### Scenario: Policy has mapping documents
- **WHEN** the agent loads a policy and queries `mapping_documents WHERE policy_id = '<id>'`
- **THEN** the agent SHALL use the returned documents for cross-framework coverage analysis

#### Scenario: No mapping documents found
- **WHEN** the query returns zero rows
- **THEN** the agent SHALL skip cross-framework coverage analysis and state this clearly

### Requirement: Single consolidated skill file
All domain knowledge previously split across `audit-methodology`, `evidence-schema`, `coverage-mapping`, and `gemara-mcp` SHALL be consolidated into a single `skills/studio-audit/SKILL.md`.

#### Scenario: Skill content priorities
- **WHEN** the consolidated skill is authored
- **THEN** the skill SHALL lead with: classification criteria (Strength/Finding/Gap/Observation), satisfaction determination, and coverage-mapping rules
- **THEN** the skill SHALL end with: compact table reference
- **THEN** the skill SHALL NOT contain SQL query templates

### Requirement: Few-shot examples preserved
The `prompts/few-shot/*.yaml` classification examples SHALL remain unchanged.

#### Scenario: Few-shot loading
- **WHEN** the agent starts
- **THEN** `load_few_shot_examples()` SHALL load all YAML files from `prompts/few-shot/`
- **THEN** the examples SHALL appear in the system prompt
