## ADDED Requirements

### Requirement: Editor is the default view
The workbench SHALL display the workspace editor as the default view when the application loads.

#### Scenario: Fresh load
- **WHEN** the user opens the workbench and authentication completes
- **THEN** the workspace editor view is displayed with an empty CodeMirror YAML editor and the toolbar (Validate, Save, Publish, Import)

#### Scenario: Navigation from missions
- **WHEN** the user navigates from the missions list back to the workspace
- **THEN** the editor retains its current content (no reset)

### Requirement: Editor toolbar provides core actions
The workspace editor toolbar SHALL include Validate, Save, Publish, and Import actions.

#### Scenario: Validate action
- **WHEN** the user clicks "Validate" with YAML content in the editor
- **THEN** the system detects the Gemara definition type and calls `/api/validate`
- **THEN** the validation result is displayed below the editor

#### Scenario: Save action
- **WHEN** the user clicks "Save" with YAML content in the editor
- **THEN** the system calls `/api/workspace/save` with the content and a filename derived from the artifact type
- **THEN** a success or error message is displayed

#### Scenario: Publish action
- **WHEN** the user clicks "Publish"
- **THEN** the publish dialog opens with the current editor content as the artifact

#### Scenario: Import action
- **WHEN** the user clicks "Import"
- **THEN** the registry import dialog opens as a modal overlay

### Requirement: Mission artifacts populate the editor
When a mission produces an artifact, the workspace editor SHALL be populated with the artifact content.

#### Scenario: Agent produces first artifact
- **WHEN** an active mission's agent produces a YAML artifact via SSE
- **THEN** the workspace editor content is replaced with the artifact YAML
- **THEN** the editor filename and definition type update to match the artifact

#### Scenario: Agent produces subsequent artifact
- **WHEN** the editor already contains content and the agent produces a new artifact
- **THEN** the editor content is replaced with the new artifact
- **THEN** the previous content is recoverable via CodeMirror undo (Ctrl+Z)

### Requirement: Chat drawer for active missions
The workspace view SHALL display a chat drawer when a mission is active.

#### Scenario: Mission started
- **WHEN** the user starts a new mission from the missions view
- **THEN** the workspace view navigates to the editor
- **THEN** the chat drawer slides open on the right showing the ChatPanel
- **THEN** the agent's streaming messages appear in the chat drawer

#### Scenario: No active mission
- **WHEN** no mission has an active status (submitted, working, input-required)
- **THEN** the chat drawer is hidden
- **THEN** the editor occupies the full width

#### Scenario: User dismisses chat
- **WHEN** the user closes the chat drawer during an active mission
- **THEN** the drawer hides and the editor expands to full width
- **THEN** the mission continues running in the background
- **THEN** a "Chat" button appears in the toolbar to re-open the drawer

#### Scenario: User replies in chat
- **WHEN** the mission status is "input-required" and the user types in the chat drawer
- **THEN** the reply is sent to the agent via `sendReply` using the mission's stored `agentName`

### Requirement: Editor state persists across interactions
The workspace editor content SHALL persist in shared state (Preact signals) accessible by all components.

#### Scenario: Import updates editor
- **WHEN** the registry import dialog injects a mapping-reference
- **THEN** the editor content reflects the injected reference immediately

#### Scenario: Editor content survives navigation
- **WHEN** the user navigates to the missions list and back to the workspace
- **THEN** the editor content is unchanged
