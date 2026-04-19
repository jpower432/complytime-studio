## ADDED Requirements

### Requirement: Publish dialog accepts workspace artifacts
The publish dialog SHALL default to including all workspace artifacts, not just the active editor content.

#### Scenario: Multiple workspace artifacts
- **WHEN** the user opens the publish dialog with 3 artifacts in the workspace
- **THEN** all 3 artifacts are listed with checkboxes, all checked by default
- **THEN** the user can uncheck artifacts to exclude them

#### Scenario: Single artifact workspace
- **WHEN** the workspace contains exactly one artifact
- **THEN** the publish dialog shows that artifact checked
- **THEN** the behavior is identical to the current single-artifact publish

#### Scenario: Empty workspace
- **WHEN** the workspace is empty and the user clicks Publish
- **THEN** the Publish button is disabled with a tooltip "No artifacts to publish"

### Requirement: Bundle includes all selected artifacts
The publish request SHALL include all checked workspace artifacts in the `artifacts[]` payload.

#### Scenario: Publish selected subset
- **WHEN** the user unchecks 1 of 3 artifacts and clicks Publish
- **THEN** the POST to `/api/publish` includes only the 2 checked artifacts
- **THEN** the resulting OCI bundle contains 2 layers

#### Scenario: Publish all
- **WHEN** the user leaves all artifacts checked and clicks Publish
- **THEN** the POST includes all workspace artifacts
