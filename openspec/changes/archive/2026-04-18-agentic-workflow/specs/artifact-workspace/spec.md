## ADDED Requirements

### Requirement: Workspace store holds multiple artifacts
The workbench SHALL maintain a workspace store (`workspace.ts`) that holds zero or more named artifacts simultaneously, keyed by artifact name.

#### Scenario: Empty workspace on first load
- **WHEN** the user opens the workbench for the first time (no localStorage data)
- **THEN** the workspace contains zero artifacts
- **THEN** the editor displays an empty YAML editor

#### Scenario: Artifact added to workspace
- **WHEN** an artifact is added to the workspace (via import, apply proposal, or direct creation)
- **THEN** the workspace contains the artifact keyed by its name
- **THEN** the artifact appears as a tab in the tab bar

#### Scenario: Duplicate name overwrites
- **WHEN** an artifact is added with a name that already exists in the workspace
- **THEN** the existing artifact's content is replaced with the new content
- **THEN** no duplicate tab is created

### Requirement: One artifact is active at a time
The workspace SHALL track exactly one "active" artifact whose content is displayed in the CodeMirror editor.

#### Scenario: Activate by tab click
- **WHEN** the user clicks a tab in the artifact tab bar
- **THEN** the clicked artifact becomes active
- **THEN** the editor content updates to show the artifact's YAML
- **THEN** the definition dropdown updates to the artifact's definition type

#### Scenario: New artifact auto-activates
- **WHEN** an artifact is added to the workspace
- **THEN** the new artifact becomes the active artifact

#### Scenario: Active artifact removed
- **WHEN** the user closes the active artifact's tab
- **THEN** the workspace activates the next artifact (or previous if last)
- **THEN** if no artifacts remain, the editor is empty

### Requirement: Tab bar displays workspace artifacts
The workspace editor SHALL display a horizontal tab bar above the CodeMirror editor showing all workspace artifacts.

#### Scenario: Tab rendering
- **WHEN** the workspace contains artifacts
- **THEN** each artifact is shown as a tab with its name
- **THEN** the active artifact's tab is visually distinguished

#### Scenario: Tab close
- **WHEN** the user clicks the close button on a tab
- **THEN** the artifact is removed from the workspace
- **THEN** the tab disappears from the tab bar

#### Scenario: Tab overflow
- **WHEN** the workspace contains more tabs than fit in the visible area
- **THEN** the tab bar scrolls horizontally

### Requirement: Workspace persists in localStorage
The workspace state SHALL persist to `localStorage` and re-hydrate on page load.

#### Scenario: Page refresh preserves artifacts
- **WHEN** the user refreshes the page with artifacts in the workspace
- **THEN** all artifacts are restored from localStorage
- **THEN** the previously active artifact is re-activated

#### Scenario: Storage capacity warning
- **WHEN** the workspace approaches 80% of localStorage capacity
- **THEN** a non-blocking warning is displayed to the user

### Requirement: Editor signals remain backward-compatible
The existing `editorContent`, `editorFilename`, and `editorDefinition` signals SHALL continue to function as views of the active workspace artifact.

#### Scenario: Existing component reads editor signals
- **WHEN** a component reads `editorContent.value`
- **THEN** it receives the active workspace artifact's YAML content

#### Scenario: Existing component writes editor content
- **WHEN** a component sets `editorContent.value`
- **THEN** the active workspace artifact's content is updated
