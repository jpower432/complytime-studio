## ADDED Requirements

### Requirement: Gap Analyst agent exists as an ADK llmagent
The system SHALL provide a Gap Analyst specialist agent implemented as a Go ADK `llmagent` with its own A2A server, following the same pattern as the Threat Modeler and Policy Composer.

#### Scenario: Agent starts on dedicated port
- **WHEN** the Studio binary starts
- **THEN** the Gap Analyst A2A server SHALL listen on its configured port (default 8002) and serve an agent card at `/.well-known/agent.json`

#### Scenario: Agent is registered as orchestrator sub-agent
- **WHEN** the orchestrator initializes
- **THEN** the Gap Analyst SHALL be included in the orchestrator's sub-agent list for delegation

### Requirement: Gap Analyst consumes MappingDocument as input
The system SHALL require a validated Gemara MappingDocument as input to the Gap Analyst. The agent SHALL NOT author MappingDocuments.

#### Scenario: MappingDocument provided in delegation
- **WHEN** the orchestrator delegates to the Gap Analyst with a MappingDocument included in the message
- **THEN** the Gap Analyst SHALL parse the MappingDocument and use it as the basis for gap analysis

#### Scenario: No MappingDocument provided
- **WHEN** the orchestrator delegates to the Gap Analyst without a MappingDocument
- **THEN** the Gap Analyst SHALL respond requesting a MappingDocument before proceeding

#### Scenario: Invalid MappingDocument provided
- **WHEN** the Gap Analyst receives a MappingDocument that fails gemara-mcp validation
- **THEN** the agent SHALL report the validation errors and request a corrected MappingDocument

### Requirement: Gap Analyst produces AuditLog artifacts
The system SHALL produce Gemara `#AuditLog` artifacts as the primary output of gap analysis, classifying each target reference entry by coverage.

#### Scenario: Full coverage analysis
- **WHEN** the Gap Analyst completes analysis of a MappingDocument
- **THEN** it SHALL produce an AuditLog with one `AuditResult` per target reference entry, each classified as `Gap`, `Finding`, `Observation`, or `Strength`

#### Scenario: Gap classification
- **WHEN** a target reference entry has relationship `no-match` or is absent from the MappingDocument
- **THEN** the corresponding AuditResult SHALL have `type: "Gap"` with a recommendation for remediation

#### Scenario: Partial coverage classification
- **WHEN** a target reference entry has mappings with relationship `supports` or low strength (1-4)
- **THEN** the corresponding AuditResult SHALL have `type: "Finding"` with evidence referencing the mapping entries

#### Scenario: Strong coverage classification
- **WHEN** a target reference entry has mappings with relationship `implements` or `equivalent` and high strength (7-10)
- **THEN** the corresponding AuditResult SHALL have `type: "Strength"`

### Requirement: Gap Analyst validates output before returning
The system SHALL validate the produced AuditLog via gemara-mcp `validate_gemara_artifact` with definition `#AuditLog` before returning to the orchestrator.

#### Scenario: Validation passes
- **WHEN** the AuditLog passes validation
- **THEN** the agent SHALL return the validated YAML to the orchestrator

#### Scenario: Validation fails
- **WHEN** the AuditLog fails validation
- **THEN** the agent SHALL fix errors and re-validate up to 3 attempts before reporting failure

### Requirement: Gap Analyst uses gemara-mcp and github-mcp tools
The system SHALL connect the Gap Analyst to gemara-mcp (for validation) and github-mcp (for fetching reference context) via MCP toolsets.

#### Scenario: MCP tools available
- **WHEN** the Gap Analyst initializes with MCP transport configured
- **THEN** it SHALL have access to `validate_gemara_artifact`, `migrate_gemara_artifact` from gemara-mcp and `get_file_contents`, `search_code`, `search_repositories` from github-mcp

### Requirement: Gap Analyst A2A skills
The system SHALL expose gap analysis as an A2A skill for external invocation.

#### Scenario: Skill advertised in agent card
- **WHEN** a client queries the Gap Analyst's agent card
- **THEN** the card SHALL include a `gap-analysis` skill with description, tags, and examples
