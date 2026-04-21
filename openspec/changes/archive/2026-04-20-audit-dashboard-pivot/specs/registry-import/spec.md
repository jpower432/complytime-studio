## MODIFIED Requirements

### Requirement: Import artifacts into policy store
The registry import flow SHALL store imported Policy artifacts in ClickHouse instead of the browser workspace. The import dialog SHALL validate the artifact type and only accept Policies and MappingDocuments.

#### Scenario: Import Policy from OCI
- **WHEN** the user enters an OCI reference in the import dialog and the artifact is a valid Policy
- **THEN** the system pulls the artifact, validates via gemara-mcp, and stores it in the ClickHouse `policies` table

#### Scenario: Import MappingDocument from OCI
- **WHEN** the user enters an OCI reference and the artifact is a valid MappingDocument
- **THEN** the system stores it in the ClickHouse `mapping_documents` table and prompts the user to link it to an existing policy

#### Scenario: Reject non-audit artifacts
- **WHEN** the imported artifact is a ThreatCatalog, ControlCatalog, or other non-audit type
- **THEN** the system displays a message: "Studio imports Policies and MappingDocuments only. Author artifacts using your local toolchain."
