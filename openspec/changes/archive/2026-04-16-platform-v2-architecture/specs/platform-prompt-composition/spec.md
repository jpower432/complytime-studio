## ADDED Requirements

### Requirement: Shared platform prompt file
A file `agents/platform.md` SHALL exist containing the shared identity, constraints, and rules that apply to all ComplyTime Studio agents.

#### Scenario: Platform identity content
- **WHEN** `agents/platform.md` is read
- **THEN** it contains: agent identity ("You are a ComplyTime Studio specialist agent"), validation rules ("Always validate via gemara-mcp"), content integrity rules ("Never fabricate repository content"), and scope boundaries ("Do not interact with OCI registries")

### Requirement: Helm renders platform prompt as ConfigMap
Helm SHALL render `agents/platform.md` into a ConfigMap (`studio-platform-prompts`) with keys that can be referenced by kagent's `promptTemplate` include function.

#### Scenario: ConfigMap generation
- **WHEN** `helm template` is run
- **THEN** a ConfigMap named `studio-platform-prompts` is created with at least keys `identity` and `constraints` derived from `agents/platform.md`

### Requirement: Agent CRDs use promptTemplate for composition
Each Declarative Agent CRD SHALL use `promptTemplate.dataSources` to reference the `studio-platform-prompts` ConfigMap and compose the systemMessage from platform fragments plus agent-specific content.

#### Scenario: Prompt template composition
- **WHEN** an Agent CRD is rendered for `studio-threat-modeler`
- **THEN** the `systemMessage` field includes `{{include "platform/identity"}}` and `{{include "platform/constraints"}}` followed by the agent-specific prompt content
- **THEN** kagent resolves the template at reconciliation time into a fully expanded system message

### Requirement: Agent prompt.md contains only workflow
After platform constraints are extracted, each agent's `prompt.md` SHALL contain only the agent-specific workflow, interaction style, and output format. It SHALL NOT duplicate platform-level rules.

#### Scenario: Slimmed threat modeler prompt
- **WHEN** `agents/threat-modeler/prompt.md` is read
- **THEN** it does NOT contain "Always validate after authoring" (moved to platform.md)
- **THEN** it does NOT contain "Never fabricate repository content" (moved to platform.md)
- **THEN** it DOES contain the STRIDE workflow steps specific to this agent
