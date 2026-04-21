## ADDED Requirements

### Requirement: Import policy from OCI registry
The system SHALL allow users to import a Policy artifact from an OCI registry reference and store it in ClickHouse with metadata (policy_id, title, version, import timestamp, OCI reference).

#### Scenario: Successful policy import
- **WHEN** the user provides a valid OCI reference containing a Gemara Policy artifact
- **THEN** the system pulls the artifact, validates it via gemara-mcp using definition `#Policy`, and stores it in the `policies` ClickHouse table

#### Scenario: Invalid artifact
- **WHEN** the imported artifact fails gemara-mcp validation
- **THEN** the system displays the validation errors and does not store the artifact

### Requirement: List stored policies
The system SHALL display all stored policies with title, version, import date, and linked MappingDocument count in the Policies view.

#### Scenario: Policies view
- **WHEN** the user navigates to the Policies view
- **THEN** the system queries ClickHouse and displays a table of stored policies sorted by import date descending

### Requirement: Policy detail view
The system SHALL display the full YAML content of a stored policy with metadata and linked MappingDocuments.

#### Scenario: View policy detail
- **WHEN** the user selects a policy from the Policies list
- **THEN** the system displays the policy YAML in a read-only viewer with tabs for metadata, scope, imports, adherence, and linked MappingDocuments
