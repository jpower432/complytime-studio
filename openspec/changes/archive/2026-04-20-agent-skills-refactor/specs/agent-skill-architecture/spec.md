## ADDED Requirements

### Requirement: Prompt contains workflow only
The assistant `prompt.md` SHALL contain only workflow steps (gather inputs, query, classify, author, validate, return), tool awareness (which MCP servers to use and when), input/output format, and behavioral constraints. Domain knowledge (schemas, classification tables, query patterns, audit methodology) SHALL NOT appear in the prompt.

#### Scenario: Prompt references skills instead of inlining knowledge
- **WHEN** the assistant prompt is loaded
- **THEN** it contains no SQL queries, no classification tables, no frequency-to-cycle mappings, and no mapping strength interpretation rules
- **THEN** it references skills by name for domain knowledge (e.g., "load evidence-schema skill for table structure")

#### Scenario: Prompt is under 50 lines
- **WHEN** the assistant prompt.md is measured
- **THEN** it contains fewer than 50 non-blank lines

### Requirement: Internal skills follow SKILL.md format
Each internal skill under `skills/<name>/SKILL.md` SHALL include frontmatter with `name` and `description` fields, followed by domain knowledge content. Skills SHALL contain knowledge, not workflow.

#### Scenario: Skill file structure
- **WHEN** a skill file is read
- **THEN** it begins with YAML frontmatter (`---`) containing `name` and `description`
- **THEN** the body contains domain knowledge without step-by-step workflow instructions

### Requirement: gemara-mcp skill covers MCP tools and resources
The `skills/gemara-mcp/SKILL.md` SHALL document the Gemara layer model (L1-L7), available MCP tools (`validate_gemara_artifact`, `migrate_gemara_artifact`), available MCP resources (`gemara://lexicon`, `gemara://schema/definitions`), the validation workflow (author, validate, fix, re-validate max 3), and which layers the assistant produces (L7) vs consumes (L3, L5, L6).

#### Scenario: Agent knows how to validate an artifact
- **WHEN** the agent needs to validate authored YAML
- **THEN** it uses `validate_gemara_artifact` with the appropriate definition name (e.g., `#AuditLog`)
- **THEN** it follows the fix-and-retry pattern up to 3 attempts

#### Scenario: Agent can access schema definitions
- **WHEN** the agent needs to understand a Gemara schema shape
- **THEN** it reads the `gemara://schema/definitions` resource via MCP

### Requirement: evidence-schema skill documents ClickHouse tables
The `skills/evidence-schema/SKILL.md` SHALL document all ClickHouse table schemas (`evidence`, `policies`, `mapping_documents`, `audit_logs`), column types, enum values, and standard query patterns (target inventory, per-target evidence, cadence validation).

#### Scenario: Agent constructs evidence query from schema knowledge
- **WHEN** the agent needs target inventory for an audit
- **THEN** it constructs a query using column names and types documented in the skill
- **THEN** it uses `run_select_query` via clickhouse-mcp to execute it

### Requirement: audit-methodology skill encodes assessment rules
The `skills/audit-methodology/SKILL.md` SHALL document assessment cadence rules (frequency-to-cycle-count mapping), missing cycle classification (cadence gap = Finding), finding vs gap vs observation vs strength classification criteria, and satisfaction determination levels.

#### Scenario: Agent classifies a missing assessment cycle
- **WHEN** evidence queries reveal a gap in the expected assessment cadence
- **THEN** the agent classifies it as a Finding per the audit-methodology skill
- **THEN** the agent documents the specific missing dates

### Requirement: coverage-mapping skill encodes cross-framework logic
The `skills/coverage-mapping/SKILL.md` SHALL document cross-framework join logic (AuditResult to MappingDocument matching), strength/confidence interpretation table, coverage status derivation rules, multi-mapping resolution (strongest wins), and coverage matrix presentation format.

#### Scenario: Agent derives framework coverage from mapping strength
- **WHEN** an AuditResult maps to an external framework entry with strength 8 and high confidence
- **THEN** the agent classifies coverage as "Covered" per the coverage-mapping skill

### Requirement: External skills loaded from rhaml-23/prompt
The `agent.yaml` SHALL reference `skills/research.md` and `skills/gemara.md` from `rhaml-23/prompt` as external gitRef skills.

#### Scenario: Research synthesis skill is available
- **WHEN** the agent produces output containing compliance determinations
- **THEN** it applies source hierarchy and confidence flagging from the research skill

### Requirement: Dead agents deleted
The directories `agents/threat-modeler/` and `agents/policy-composer/` SHALL be removed. The `agents/platform.md` file SHALL be removed. The ConfigMap SHALL only include the assistant prompt.

#### Scenario: No unused agent artifacts remain
- **WHEN** the agents directory is listed
- **THEN** only `agents/assistant/` exists (no `threat-modeler/`, `policy-composer/`, or `platform.md`)

#### Scenario: ConfigMap contains assistant only
- **WHEN** the agent-prompts-configmap template is rendered
- **THEN** it contains only the `assistant` key
