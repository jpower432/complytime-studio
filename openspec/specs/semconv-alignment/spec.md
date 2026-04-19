## ADDED Requirements

### Requirement: Semconv-to-ClickHouse column mapping is documented
The change SHALL document the mapping between every `beacon.evidence` semconv attribute and its corresponding ClickHouse `evidence` table column.

#### Scenario: All existing semconv attributes have a column mapping
- **WHEN** the mapping document is reviewed
- **THEN** every attribute in `registry.policy` and `registry.compliance` groups maps to exactly one ClickHouse column
- **THEN** data types are compatible (OTel string → ClickHouse String, OTel enum → ClickHouse Enum8, etc.)

#### Scenario: Column names follow a consistent convention
- **WHEN** an OTel attribute name is mapped to a ClickHouse column
- **THEN** the column name uses snake_case derived from the attribute path (e.g., `compliance.control.id` → `control_id`)

### Requirement: Gemara-specific semconv gaps are identified and proposed
The change SHALL document attributes required for Gemara audit-grade evidence that are missing from the current `beacon.evidence` entity.

#### Scenario: Missing attributes documented
- **WHEN** the semconv gap analysis is reviewed
- **THEN** it lists `compliance.policy.id`, `compliance.assessment.requirement.id`, `compliance.assessment.plan.id`, `compliance.assessment.confidence`, and `compliance.assessment.steps` as required additions

#### Scenario: Each proposed attribute has a type, group, and rationale
- **WHEN** a proposed attribute is reviewed
- **THEN** it specifies the OTel type, the target attribute group (`registry.compliance`), and a rationale for inclusion

### Requirement: Enrichment provenance is tracked
The `compliance.enrichment.status` attribute SHALL indicate how compliance context was populated.

#### Scenario: Gemara-native signal (Path A)
- **WHEN** complyctl/ProofWatch emits a signal with full `compliance.*` attributes
- **THEN** `compliance.enrichment.status` is `Success` or absent

#### Scenario: Truthbeam enrichment succeeds (Path B)
- **WHEN** truthbeam maps a `policy.rule.id` to a control and populates `compliance.*` attributes
- **THEN** `compliance.enrichment.status` is `Success`

#### Scenario: Truthbeam enrichment partially succeeds (Path B)
- **WHEN** truthbeam maps some but not all compliance attributes
- **THEN** `compliance.enrichment.status` is `Partial`

#### Scenario: Truthbeam enrichment fails (Path B)
- **WHEN** truthbeam cannot find a mapping for the `policy.rule.id`
- **THEN** `compliance.enrichment.status` is `Unmapped`
