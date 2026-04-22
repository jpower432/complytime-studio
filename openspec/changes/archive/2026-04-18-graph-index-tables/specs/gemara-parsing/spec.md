## ADDED Requirements

### Requirement: ParseControlCatalog function
The system SHALL export a `ParseControlCatalog(content string, catalogID string, policyID string) ([]ControlRow, []AssessmentRequirementRow, []ControlThreatRow, error)` function in `internal/gemara/` that parses Gemara `#ControlCatalog` YAML using `go-gemara` types and returns flat rows for `controls`, `assessment_requirements`, and `control_threats` tables.

#### Scenario: Valid ControlCatalog with controls, requirements, and threat references
- **WHEN** `ParseControlCatalog` receives valid `#ControlCatalog` YAML with 2 controls, each with 2 assessment requirements, and the first control referencing 1 threat
- **THEN** it SHALL return 2 `ControlRow`, 4 `AssessmentRequirementRow`, and 1 `ControlThreatRow` with correct field values

#### Scenario: Invalid YAML content
- **WHEN** `ParseControlCatalog` receives content that fails `goccy/go-yaml` unmarshalling
- **THEN** it SHALL return a non-nil error wrapping the parse failure

#### Scenario: Catalog with no controls
- **WHEN** `ParseControlCatalog` receives valid YAML with an empty `controls` array
- **THEN** it SHALL return empty slices and no error

### Requirement: ParseThreatCatalog function
The system SHALL export a `ParseThreatCatalog(content string, catalogID string, policyID string) ([]ThreatRow, error)` function in `internal/gemara/` that parses Gemara `#ThreatCatalog` YAML using `go-gemara` types and returns flat rows for the `threats` table.

#### Scenario: Valid ThreatCatalog with threats
- **WHEN** `ParseThreatCatalog` receives valid `#ThreatCatalog` YAML with 3 threats
- **THEN** it SHALL return 3 `ThreatRow` with correct `ThreatID`, `Title`, `Description`, and `GroupID`

#### Scenario: Invalid YAML content
- **WHEN** `ParseThreatCatalog` receives content that fails unmarshalling
- **THEN** it SHALL return a non-nil error wrapping the parse failure

### Requirement: Row types defined in gemara package
The `ControlRow`, `AssessmentRequirementRow`, `ControlThreatRow`, and `ThreatRow` structs SHALL be defined in `internal/gemara/`. The `internal/store/` package SHALL reference these types for database insert operations.

#### Scenario: Store insert uses gemara types
- **WHEN** `InsertControls` is called
- **THEN** it SHALL accept `[]gemara.ControlRow` (from `internal/gemara`)
