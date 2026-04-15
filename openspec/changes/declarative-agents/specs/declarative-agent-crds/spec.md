## ADDED Requirements

### Requirement: Canonical agent definition files

Each specialist agent SHALL be defined in `agents/<name>/agent.yaml` (identity, MCP tools, A2A skills) and `agents/<name>/prompt.md` (system prompt). These files are the source of truth for agent configuration.

#### Scenario: Agent directory structure

- **WHEN** the repository is inspected
- **THEN** `agents/threat-modeler/`, `agents/gap-analyst/`, and `agents/policy-composer/` each contain `agent.yaml` and `prompt.md`

#### Scenario: agent.yaml contains portable fields only

- **WHEN** `agent.yaml` is inspected
- **THEN** it contains `name`, `description`, `prompt` (file reference), `model` (provider + name), `mcp` (server names + tool lists), and `a2a.skills` — no kagent-specific fields

### Requirement: Declarative Agent CRDs rendered from canonical definitions

The Helm chart SHALL render kagent Declarative Agent CRDs from the canonical agent definitions. Each agent CRD SHALL use `runtime: go`, reference a shared `ModelConfig`, and load its prompt from a ConfigMap.

#### Scenario: Threat modeler rendered as Declarative Agent

- **WHEN** `helm template` is run
- **THEN** a kagent `Agent` CRD is rendered with `type: Declarative`, `runtime: go`, `modelConfig: studio-model`, and `systemMessageFrom` referencing the `studio-agent-prompts` ConfigMap

#### Scenario: MCP tool filtering preserved

- **WHEN** the threat modeler Agent CRD is rendered
- **THEN** its `tools` array contains entries for `studio-gemara-mcp` with `toolNames: [validate_gemara_artifact, migrate_gemara_artifact]` and `studio-github-mcp` with `toolNames: [get_file_contents, search_code, search_repositories]`

#### Scenario: Gap analyst includes ClickHouse MCP

- **WHEN** the gap analyst Agent CRD is rendered with `clickhouse.enabled=true`
- **THEN** its `tools` array includes an entry for `studio-clickhouse-mcp` (backed by `ClickHouse/mcp-clickhouse`) with `toolNames: [run_select_query, list_databases, list_tables]`

### Requirement: Shared ModelConfig CRD

The Helm chart SHALL render a single kagent `ModelConfig` CRD referenced by all Declarative agents. Provider and model name SHALL be configurable via `values.yaml`.

#### Scenario: ModelConfig rendered from values

- **WHEN** `values.yaml` sets `model.provider: AnthropicVertexAI` and `model.name: claude-sonnet-4-20250514`
- **THEN** a `ModelConfig` named `studio-model` is rendered with those values

### Requirement: Agent prompts in ConfigMap

Agent prompts SHALL be rendered into a ConfigMap (`studio-agent-prompts`) from the `agents/<name>/prompt.md` files. Each key SHALL be the agent name.

#### Scenario: Prompt ConfigMap generated

- **WHEN** `helm template` is run
- **THEN** a ConfigMap named `studio-agent-prompts` exists with keys `threat-modeler`, `gap-analyst`, and `policy-composer`, each containing the corresponding `prompt.md` content

### Requirement: No BYO agent binary

The system SHALL NOT include a `cmd/agents/` binary, `internal/agents/` package, or agents Docker image. All agent wiring is handled by kagent Declarative CRDs.

#### Scenario: Agent Go code deleted

- **WHEN** the codebase is inspected
- **THEN** `cmd/agents/` and `internal/agents/` do not exist

### Requirement: No orchestrator

The system SHALL NOT include an orchestrator agent or routing skill. Users select specialists directly via the platform UI or agent directory API.

#### Scenario: Orchestrator artifacts deleted

- **WHEN** the codebase is inspected
- **THEN** `agents/orchestrator.md` and `skills/orchestrator-routing/` do not exist
