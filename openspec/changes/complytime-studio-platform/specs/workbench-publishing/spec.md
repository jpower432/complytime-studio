## ADDED Requirements

### Requirement: Publish action in artifact panel
The workbench SHALL provide a "Publish" button in the artifact panel toolbar that initiates the OCI bundle publishing workflow.

#### Scenario: User clicks Publish with valid artifacts
- **WHEN** a mission has one or more validated artifacts and the user clicks "Publish"
- **THEN** the workbench SHALL display a publish dialog requesting the target registry reference

#### Scenario: Publish button disabled without artifacts
- **WHEN** a mission has no artifacts
- **THEN** the "Publish" button SHALL be disabled

### Requirement: Publish dialog with registry target
The workbench SHALL display a publish dialog where the user confirms the registry reference, selected artifacts, and signing preference before triggering the push.

#### Scenario: Dialog pre-fills from metadata
- **WHEN** the publish dialog opens
- **THEN** the registry reference field SHALL pre-fill from artifact metadata (id + version) if available

#### Scenario: User overrides registry reference
- **WHEN** the user edits the registry reference field
- **THEN** the override SHALL be used instead of the metadata-derived reference

#### Scenario: User confirms publish
- **WHEN** the user clicks "Publish" in the dialog
- **THEN** the workbench SHALL send the artifacts to the orchestrator's publish workflow and display progress

#### Scenario: User cancels publish
- **WHEN** the user clicks "Cancel" in the dialog
- **THEN** no publish action SHALL occur

### Requirement: Publish progress and result feedback
The workbench SHALL display the publishing outcome to the user.

#### Scenario: Publish succeeds
- **WHEN** the publish_bundle tool returns successfully
- **THEN** the workbench SHALL display the OCI reference, manifest digest, and signature digest

#### Scenario: Publish fails
- **WHEN** the publish_bundle tool returns an error
- **THEN** the workbench SHALL display the error message to the user
