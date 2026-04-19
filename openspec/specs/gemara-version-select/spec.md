## ADDED Requirements

### Requirement: Gemara version input in toolbar
The artifact toolbar SHALL include a text input for specifying the Gemara schema version used during validation.

#### Scenario: Default version
- **WHEN** the user opens the workspace with no prior version selection
- **THEN** the version input SHALL display `latest`

#### Scenario: User specifies a version
- **WHEN** the user types a version string (e.g., `0.20.0`) into the version input
- **THEN** the system SHALL store that version for the active artifact
- **THEN** subsequent Validate calls SHALL use the specified version

#### Scenario: Invalid version
- **WHEN** the user specifies a version that does not exist in the CUE registry
- **AND** clicks Validate
- **THEN** the validation result bar SHALL display the error returned by gemara-mcp

### Requirement: Version stored per artifact
Each artifact in the workspace SHALL have an independent `gemaraVersion` field.

#### Scenario: Switching artifacts preserves version
- **WHEN** the user sets artifact A to version `0.20.0` and artifact B to `latest`
- **AND** switches between tabs
- **THEN** the version input SHALL reflect the active artifact's stored version

#### Scenario: New artifact defaults to latest
- **WHEN** the user creates a new artifact (via "+" or paste)
- **THEN** the artifact's `gemaraVersion` SHALL default to `latest`
