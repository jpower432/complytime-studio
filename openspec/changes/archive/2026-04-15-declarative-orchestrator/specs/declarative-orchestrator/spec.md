## ADDED Requirements

### Requirement: Orchestrator deployed as Declarative Agent CRD
The Helm chart SHALL deploy the orchestrator as a kagent `Agent` resource with `spec.type: Declarative` and `spec.declarative.runtime: go`.

#### Scenario: Chart renders Declarative Agent
- **WHEN** `helm template` is run with default values
- **THEN** the output SHALL contain an `Agent` resource named `studio-orchestrator` with `spec.type: Declarative`
- **THEN** the resource SHALL specify `spec.declarative.runtime: go`

#### Scenario: Orchestrator pod managed by kagent
- **WHEN** the `studio-orchestrator` Agent CRD is applied
- **THEN** the kagent controller SHALL create a Deployment using the Go ADK runtime image
- **THEN** the pod SHALL reach `Ready` status within 120 seconds

### Requirement: Orchestrator references specialists as agent tools
The declarative orchestrator Agent CRD SHALL include `type: Agent` tool entries for each specialist agent (threat modeler, gap analyst, policy composer).

#### Scenario: Specialist agent tools declared
- **WHEN** `helm template` is run with default values
- **THEN** the orchestrator's `spec.declarative.tools` array SHALL contain entries with `type: Agent` referencing `studio-threat-modeler`, `studio-gap-analyst`, and `studio-policy-composer`

#### Scenario: Orchestrator delegates to specialist via A2A
- **WHEN** a user sends a threat modeling request to the orchestrator
- **THEN** the orchestrator SHALL invoke the `studio-threat-modeler` agent tool via A2A
- **THEN** the specialist's response SHALL be returned to the orchestrator for further processing

### Requirement: Orchestrator references MCP servers as McpServer tools
The declarative orchestrator Agent CRD SHALL include `type: McpServer` tool entries for oras-mcp with a tool name filter.

#### Scenario: MCP server tools declared
- **WHEN** `helm template` is run with oras MCP server enabled
- **THEN** the orchestrator's `spec.declarative.tools` array SHALL contain an entry with `type: McpServer` referencing `studio-oras-mcp`
- **THEN** the McpServer tool entry SHALL specify `toolNames` limited to: `list_wellknown_registries`, `list_repositories`, `list_tags`, `list_referrers`, `fetch_manifest`, `parse_reference`

### Requirement: Orchestrator uses shared ModelConfig
The Helm chart SHALL deploy a `ModelConfig` CRD that the declarative orchestrator references via `spec.declarative.modelConfig`.

#### Scenario: ModelConfig rendered and referenced
- **WHEN** `helm template` is run with default values
- **THEN** the output SHALL contain a `ModelConfig` resource named `studio-model-config`
- **THEN** the `studio-orchestrator` Agent's `spec.declarative.modelConfig` SHALL reference `studio-model-config`

#### Scenario: Model change propagates to orchestrator
- **WHEN** the `studio-model-config` resource is updated with a different model name
- **THEN** the kagent controller SHALL restart the orchestrator pod to pick up the new model

### Requirement: Orchestrator loads skills from Git
The orchestrator Agent CRD SHALL configure `spec.skills.gitRefs` to load skill files from the complytime-studio repository.

#### Scenario: Skills init container clones repository
- **WHEN** the orchestrator pod starts
- **THEN** a `skills-init` init container SHALL clone the configured Git repository
- **THEN** skill files SHALL be mounted under `/skills/` in the runtime container

#### Scenario: Orchestrator discovers skills at startup
- **WHEN** the orchestrator runtime initializes
- **THEN** skill descriptions from `SKILL.md` frontmatter SHALL be injected into the system prompt
- **THEN** the `load_skill` tool SHALL be available to the orchestrator

### Requirement: Orchestrator tracing via OpenTelemetry
The declarative orchestrator SHALL emit OpenTelemetry traces for agent invocations and tool calls when kagent's OTel integration is enabled.

#### Scenario: Traces emitted for agent-to-agent delegation
- **WHEN** the orchestrator delegates to a specialist agent via A2A
- **AND** kagent OTel tracing is enabled
- **THEN** the trace SHALL contain spans for the orchestrator's `agent_run` and the agent tool invocation
- **THEN** spans SHALL be correlated across the orchestrator and specialist agent
