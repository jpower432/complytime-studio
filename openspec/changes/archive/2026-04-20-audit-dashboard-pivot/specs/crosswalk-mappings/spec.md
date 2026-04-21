## ADDED Requirements

### Requirement: Import MappingDocument
The system SHALL allow users to import a MappingDocument artifact from an OCI registry and link it to a stored policy.

#### Scenario: Successful mapping import
- **WHEN** the user provides an OCI reference containing a Gemara MappingDocument and selects a target policy
- **THEN** the system validates the MappingDocument, stores it in ClickHouse, and links it to the selected policy

#### Scenario: Mapping references unknown policy criteria
- **WHEN** a MappingDocument references source criteria IDs not found in the linked policy
- **THEN** the system displays a warning listing unresolved references but still stores the mapping

### Requirement: View crosswalk mapping
The system SHALL display MappingDocument entries showing source (internal criteria) to target (external framework entry) with strength and confidence-level scores.

#### Scenario: Mapping detail view
- **WHEN** the user selects a MappingDocument from the policy detail view
- **THEN** the system displays a table of mapping entries with source criteria, target framework entries, strength (1-10), and confidence-level
