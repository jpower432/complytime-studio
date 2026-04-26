## MODIFIED Requirements

### Requirement: Semconv-to-ClickHouse column mapping is documented
The change SHALL document the mapping between every `beacon.evidence` semconv attribute and its corresponding ClickHouse `evidence` table column. The mapping SHALL include the new `attestation_ref` column.

#### Scenario: All existing semconv attributes have a column mapping
- **WHEN** the mapping document is reviewed
- **THEN** every attribute in `registry.policy` and `registry.compliance` groups maps to exactly one ClickHouse column

#### Scenario: Column names follow a consistent convention
- **WHEN** an OTel attribute name is mapped to a ClickHouse column
- **THEN** the column name uses snake_case derived from the attribute path (e.g., `compliance.control.id` → `control_id`)

#### Scenario: Attestation reference column mapped
- **WHEN** the mapping document is reviewed
- **THEN** `compliance.attestation.ref` maps to `attestation_ref` column of type `Nullable(String)` containing an OCI digest
