## ADDED Requirements

### Requirement: Dedicated gemara parsing package
The system SHALL provide an `internal/gemara/` Go package that centralizes all Gemara YAML parsing using `go-gemara` types and `goccy/go-yaml`. The package SHALL have no dependencies on `internal/store/` or any HTTP/database packages.

#### Scenario: Package exists with no store imports
- **WHEN** the `internal/gemara/` package is compiled
- **THEN** it SHALL NOT import `internal/store/`, `database/sql`, `net/http`, or any ClickHouse driver package

### Requirement: ParseAuditLog function
The system SHALL export a `ParseAuditLog(content string) (*AuditLogSummary, error)` function that parses Gemara `#AuditLog` YAML using the `go-gemara` `AuditLog` type and returns an `AuditLogSummary` containing `AuditStart`, `AuditEnd`, `TargetID`, `Framework`, `Strengths`, `Findings`, `Gaps`, and `Observations` counts.

#### Scenario: Valid AuditLog with mixed results
- **WHEN** `ParseAuditLog` receives valid `#AuditLog` YAML with 3 Strength, 1 Finding, 1 Gap, and 1 Observation results
- **THEN** it SHALL return `AuditLogSummary{Strengths: 3, Findings: 1, Gaps: 1, Observations: 1}` with correct `AuditStart`, `AuditEnd`, and `TargetID`

#### Scenario: Invalid YAML content
- **WHEN** `ParseAuditLog` receives content that fails `goccy/go-yaml` unmarshalling
- **THEN** it SHALL return a non-nil error wrapping the parse failure

#### Scenario: Missing required fields
- **WHEN** `ParseAuditLog` receives YAML with no `results` array
- **THEN** it SHALL return a non-nil error indicating the missing field

### Requirement: Relocate ParsePolicyContacts
The system SHALL move the existing `ParsePolicyContacts` function from `internal/store/contacts_parser.go` to `internal/gemara/contacts.go`. The function signature SHALL remain unchanged. All callers in `internal/store/` SHALL update their imports.

#### Scenario: Import path updated in handlers
- **WHEN** `internal/store/handlers.go` calls `ParsePolicyContacts`
- **THEN** it SHALL import from `internal/gemara` not `internal/store`

### Requirement: Relocate ParseMappingYAML
The system SHALL move the existing `ParseMappingYAML` function from `internal/store/mapping_parser.go` to `internal/gemara/mappings.go`. The function signature SHALL remain unchanged. All callers SHALL update their imports.

#### Scenario: Import path updated in handlers
- **WHEN** `internal/store/handlers.go` calls `ParseMappingYAML`
- **THEN** it SHALL import from `internal/gemara` not `internal/store`

### Requirement: Return types defined in gemara package
The `AuditLogSummary`, `PolicyContact`, and `MappingEntry` structs SHALL be defined in `internal/gemara/`. The `internal/store/` package SHALL reference these types for database operations.

#### Scenario: Store uses gemara types
- **WHEN** `InsertPolicyContacts` is called
- **THEN** it SHALL accept `[]gemara.PolicyContact` (from `internal/gemara`)
