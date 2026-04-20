## ADDED Requirements

### Requirement: Gemara authoring skill provides schema guidance

A skill at `skills/gemara-authoring/SKILL.md` SHALL contain minimal valid YAML skeletons and structural guidance for ThreatCatalog, ControlCatalog, RiskCatalog, and Policy artifact types.

#### Scenario: Skill contains required artifact skeletons
- **WHEN** the gemara-authoring skill is loaded by an agent
- **THEN** it SHALL include a minimal valid YAML example for each of: ThreatCatalog, ControlCatalog, RiskCatalog, Policy
- **THEN** each example SHALL include all required fields as defined by the CUE schema
- **THEN** each example SHALL include inline comments noting common validation pitfalls

#### Scenario: Skill documents cross-reference constraints
- **WHEN** an agent reads the gemara-authoring skill
- **THEN** it SHALL find documentation that `Threat.group` values MUST match a `groups[].id` in the same catalog
- **THEN** it SHALL find documentation that `Threat.capabilities` requires `metadata.mapping-references` with a CapabilityCatalog reference
- **THEN** it SHALL find documentation that `Control.threats` requires `metadata.mapping-references` with a ThreatCatalog reference

### Requirement: All specialist agents reference the authoring skill

Every specialist agent (`threat-modeler`, `gap-analyst`, `policy-composer`) SHALL include `skills/gemara-authoring` in its `agent.yaml` skills list.

#### Scenario: Agent YAML includes authoring skill
- **WHEN** `agents/threat-modeler/agent.yaml` is read
- **THEN** its `skills` array SHALL include `{ path: skills/gemara-authoring }`

#### Scenario: Helm renders authoring skill as gitRef
- **WHEN** the Helm chart is rendered
- **THEN** every Agent CRD SHALL include a `gitRefs` entry for `skills/gemara-authoring`
