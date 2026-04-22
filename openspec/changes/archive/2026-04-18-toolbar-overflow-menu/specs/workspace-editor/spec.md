## MODIFIED Requirements

### Requirement: Editor toolbar provides core actions
The workspace editor toolbar SHALL display primary actions (Validate, Publish) directly and group secondary actions (Copy YAML, Download YAML, Download All, Import) behind an overflow menu.

#### Scenario: Validate action
- **WHEN** the user clicks "Validate" with an active artifact
- **THEN** the system validates the active artifact's YAML against its definition type

#### Scenario: Publish action
- **WHEN** the user clicks "Publish"
- **THEN** the publish dialog opens with all workspace artifacts

#### Scenario: Overflow menu opens
- **WHEN** the user clicks the three-dot overflow trigger
- **THEN** a dropdown menu appears with Copy YAML, Download YAML, Download All (if >1 artifact), and Import

#### Scenario: Overflow menu closes on outside click
- **WHEN** the overflow menu is open and the user clicks outside it
- **THEN** the menu closes

#### Scenario: Download YAML action via overflow
- **WHEN** the user clicks "Download YAML" in the overflow menu
- **THEN** the browser downloads the active artifact's YAML with the artifact name as filename
- **THEN** the overflow menu closes

#### Scenario: Import action via overflow
- **WHEN** the user clicks "Import" in the overflow menu
- **THEN** the import dialog opens
- **THEN** the overflow menu closes
