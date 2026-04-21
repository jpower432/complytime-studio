## ADDED Requirements

### Requirement: Risk reasoning skill provides domain knowledge
The `skills/risk-reasoning/SKILL.md` SHALL contain domain knowledge for qualitative risk analysis: appetite vs tolerance semantics, prioritization signals, the catalog-vs-policy data boundary, and the residual risk pattern.

#### Scenario: Skill loaded by policy-composer
- **WHEN** the policy-composer agent starts a session
- **THEN** the risk-reasoning skill content SHALL be available in the agent's context alongside gemara-layers and policy-risk-linkage

### Requirement: Enrichment opt-in prompt
The policy-composer SHALL ask the user whether to include risk analysis before beginning Phase 1. Declining SHALL preserve the existing fast-path behavior.

#### Scenario: User opts in to risk enrichment
- **WHEN** the user responds affirmatively to "Do you want risk analysis included?"
- **THEN** the agent SHALL execute the enriched Phase 1 workflow (threat graph traversal, severity justification, tolerance checks)

#### Scenario: User declines risk enrichment
- **WHEN** the user declines risk enrichment
- **THEN** the agent SHALL execute the existing Phase 1 workflow unchanged (derive categories, derive entries, validate)

### Requirement: Severity justification via threat graph signals
When risk enrichment is active, each risk entry's severity SHALL be justified using countable signals from the ThreatCatalog structure.

#### Scenario: Risk with high threat density
- **WHEN** a risk links to 3+ threats with 2+ attack vectors each
- **THEN** the agent SHALL cite threat count, vector count, and capability exposure in the severity rationale
- **THEN** the severity assignment SHALL reference specific threat IDs and vector descriptions

#### Scenario: Risk with single low-exposure threat
- **WHEN** a risk links to 1 threat with 1 vector and limited capability surface
- **THEN** the agent SHALL cite the narrow exposure as rationale for a lower severity

### Requirement: Prioritization signal summary
When risk enrichment is active, the agent SHALL present a summary table of all risks with their prioritization signals before producing the RiskCatalog YAML.

#### Scenario: Summary table presented
- **WHEN** the agent completes threat graph traversal
- **THEN** the agent SHALL present a table with columns: Risk ID, Title, Severity, Threat Count, Vector Breadth, Capability Exposure, Tolerance Violation (yes/no)
- **THEN** the user SHALL have opportunity to adjust severities before YAML generation

### Requirement: Tolerance cap violation flagging
When risk enrichment is active and a RiskCategory defines `max-severity`, the agent SHALL flag risks whose severity exceeds that boundary.

#### Scenario: Risk exceeds tolerance
- **WHEN** a risk has `severity: "Critical"` in a group with `max-severity: "High"`
- **THEN** the agent SHALL flag the risk as a tolerance violation
- **THEN** the flag SHALL appear in the prioritization summary table

#### Scenario: Risk within tolerance
- **WHEN** a risk has `severity: "Medium"` in a group with `max-severity: "High"`
- **THEN** the agent SHALL NOT flag a tolerance violation

### Requirement: Impact narrative generation
When risk enrichment is active, each risk entry SHALL include an `impact` field with a narrative connecting threats to business consequence.

#### Scenario: Impact narrative for a Critical risk
- **WHEN** the agent assigns Critical severity to a risk
- **THEN** the `impact` field SHALL describe the business consequence (e.g., "Full node compromise, access to co-located workload secrets")
- **THEN** the narrative SHALL reference the specific threats and capabilities that drive the consequence

### Requirement: Residual risk identification
After Phase 2 risk-to-control linkage, the agent SHALL optionally offer to identify residual risks for partially or fully unmitigated entries.

#### Scenario: User requests residual risk catalog
- **WHEN** the risk-to-control linkage shows unmitigated or partially mitigated risks
- **AND** the user requests a residual risk catalog
- **THEN** the agent SHALL produce a separate RiskCatalog artifact for residual risks
- **THEN** the residual catalog entries SHALL reference the inherent risk catalog

#### Scenario: User declines residual risk catalog
- **WHEN** the agent offers residual risk identification
- **AND** the user declines
- **THEN** the agent SHALL proceed to Policy authoring without a residual catalog
