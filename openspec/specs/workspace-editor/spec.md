## ADDED Requirements

### Requirement: Editor is the default view
The workbench SHALL display the workspace editor as the default view when the application loads.

#### Scenario: Fresh load
- **WHEN** the user opens the workbench and authentication completes
- **THEN** the workspace editor view is displayed with an empty CodeMirror YAML editor and the toolbar (Validate, Download YAML, Publish, Import)

#### Scenario: Navigation from jobs
- **WHEN** the user navigates from the jobs list back to the workspace
- **THEN** the editor retains its current content (no reset)

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

### Requirement: Chat drawer for active jobs
The workspace view SHALL display a chat drawer when a job is active.

#### Scenario: Job started
- **WHEN** the user starts a new job from the jobs view
- **THEN** the workspace view navigates to the editor
- **THEN** the chat drawer slides open on the right showing the ChatPanel
- **THEN** the agent's streaming messages appear in the chat drawer

#### Scenario: No active job
- **WHEN** no job has an active status (submitted, working, input-required)
- **THEN** the chat drawer is hidden
- **THEN** the editor occupies the full width

#### Scenario: User dismisses chat
- **WHEN** the user closes the chat drawer during an active job
- **THEN** the drawer hides and the editor expands to full width
- **THEN** the job continues running in the background
- **THEN** a "Chat" button appears in the toolbar to re-open the drawer

#### Scenario: User replies in chat
- **WHEN** the job status is "input-required" and the user types in the chat drawer
- **THEN** the reply is sent to the agent via `sendReply` using the job's stored `agentName`

### Requirement: Editor state persists across interactions
The workspace editor content SHALL persist in shared state (Preact signals) accessible by all components.

#### Scenario: Import updates editor
- **WHEN** the registry import dialog injects a mapping-reference
- **THEN** the editor content reflects the injected reference immediately

#### Scenario: Editor content survives navigation
- **WHEN** the user navigates to the jobs list and back to the workspace
- **THEN** the editor content is unchanged
