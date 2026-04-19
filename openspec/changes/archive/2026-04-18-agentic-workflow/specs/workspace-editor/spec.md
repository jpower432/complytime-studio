## MODIFIED Requirements

### Requirement: Job artifacts populate the editor
When a job produces an artifact, the agent SHALL propose the artifact via the approval banner. On approval, the artifact is added to the workspace and activated in the editor.

#### Scenario: Agent produces first artifact
- **WHEN** an active job's agent produces a YAML artifact via SSE
- **THEN** a proposal banner appears with the artifact name
- **THEN** the workspace editor content is NOT changed until the user clicks Apply

#### Scenario: User applies proposal
- **WHEN** the user clicks Apply on the proposal banner
- **THEN** the artifact is added to the workspace (or updated if same name exists)
- **THEN** the artifact becomes the active tab
- **THEN** the editor content shows the artifact's YAML

#### Scenario: Agent produces subsequent artifact
- **WHEN** the editor already contains content and the agent produces a new artifact with a different name
- **THEN** a new proposal banner appears (replacing any existing pending proposal)
- **THEN** applying the proposal adds a new tab to the workspace without removing the previous artifact

### Requirement: Editor toolbar provides core actions
The workspace editor toolbar SHALL include Validate, Download YAML, Publish, and Import actions, each operating on the active workspace artifact.

#### Scenario: Validate action
- **WHEN** the user clicks "Validate" with an active artifact
- **THEN** the system validates the active artifact's YAML against its definition type

#### Scenario: Download YAML action
- **WHEN** the user clicks "Download YAML" with an active artifact
- **THEN** the browser downloads the active artifact's YAML with the artifact name as filename

#### Scenario: Publish action
- **WHEN** the user clicks "Publish"
- **THEN** the publish dialog opens with all workspace artifacts (not just the active one)

#### Scenario: Import action
- **WHEN** the user imports an artifact from the registry
- **THEN** the imported artifact is added to the workspace and activated
