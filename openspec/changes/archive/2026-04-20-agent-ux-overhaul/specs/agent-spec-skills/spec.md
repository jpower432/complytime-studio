## MODIFIED Requirements

### Requirement: Agent YAML supports skill references

The canonical `agent.yaml` SHALL include a `skills` array where each entry references a git-based skill pack. Internal skills (this repo) specify `path` only. External skills specify `repo`, `ref`, and `path`. All specialist agents SHALL include `skills/gemara-authoring` as an internal skill.

#### Scenario: Internal skill reference
- **WHEN** an agent.yaml contains `skills: [{ path: "skills/gemara-layers" }]`
- **THEN** Helm renders a kagent `gitRefs` entry with the platform repo URL, `ref: main`, and `path: skills/gemara-layers`

#### Scenario: Gemara authoring skill on all agents
- **WHEN** any specialist agent.yaml is read
- **THEN** its `skills` array SHALL include `{ path: skills/gemara-authoring }`

## ADDED Requirements

### Requirement: Threat-modeler runs single-shot

The threat-modeler prompt SHALL instruct the agent to run the full pipeline (gather context, analyze, author, validate, return) in one turn without asking the user to confirm intermediate results.

#### Scenario: No mid-workflow questions
- **WHEN** the threat-modeler receives a request to analyze a repository
- **THEN** the agent SHALL NOT ask the user to choose threat categories
- **THEN** the agent SHALL NOT ask the user to confirm capabilities before proceeding
- **THEN** the agent SHALL produce the ThreatCatalog and return it in one response

#### Scenario: Validation before return
- **WHEN** the threat-modeler finishes authoring a ThreatCatalog
- **THEN** the agent SHALL call `validate_gemara_artifact` with definition `#ThreatCatalog`
- **THEN** the agent SHALL fix validation errors and re-validate (max 3 attempts) before returning

### Requirement: Gap-analyst runs single-shot

The gap-analyst prompt SHALL instruct the agent to run the full audit pipeline without scope or inventory confirmation checkpoints.

#### Scenario: No confirmation exchanges
- **WHEN** the gap-analyst receives a Policy and audit timeline
- **THEN** the agent SHALL auto-derive the target inventory from ClickHouse evidence
- **THEN** the agent SHALL NOT ask the user to confirm scope, inventory, or criteria
- **THEN** the agent SHALL produce the AuditLog and summary in one response

#### Scenario: Pause only on missing prerequisites
- **WHEN** the Policy is missing, ClickHouse is unreachable, or zero evidence is found
- **THEN** the agent SHALL inform the user of the specific issue and wait for resolution

### Requirement: Policy-composer uses two-phase conversation

The policy-composer prompt SHALL use a two-phase workflow: derive-all with sensible defaults, then confirm-once before generating artifacts.

#### Scenario: Phase 1 presents derived defaults
- **WHEN** the policy-composer receives a ThreatCatalog and ControlCatalog
- **THEN** the agent SHALL derive risk categories, risk entries, scope, risk-to-control linkage, and assessment plan defaults
- **THEN** the agent SHALL present all derived values in one summary table for review

#### Scenario: Phase 2 generates on confirmation
- **WHEN** the user confirms or adjusts the derived values
- **THEN** the agent SHALL produce RiskCatalog and Policy artifacts in one response
- **THEN** both artifacts SHALL be validated with `validate_gemara_artifact` before returning

#### Scenario: Sensible defaults for unprovided fields
- **WHEN** the user does not specify RACI contacts, enforcement approach, or assessment frequency
- **THEN** the agent SHALL use defaults (artifact author for responsible, Gate/Automated enforcement, quarterly frequency)
- **THEN** the agent SHALL note all applied defaults in the output

### Requirement: Prompts do not reference MCP prompts

Agent prompts SHALL NOT instruct agents to "use gemara-mcp's `threat_assessment` prompt" or similar. MCP prompts are not callable tools. Schema guidance SHALL come from the gemara-authoring skill instead.

#### Scenario: No MCP prompt references in threat-modeler
- **WHEN** `agents/threat-modeler/prompt.md` is read
- **THEN** it SHALL NOT contain references to `threat_assessment` or `control_catalog` as MCP prompts
- **THEN** it SHALL reference the gemara-authoring skill for structural guidance
