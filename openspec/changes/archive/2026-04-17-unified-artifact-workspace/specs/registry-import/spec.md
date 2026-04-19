## ADDED Requirements

### Requirement: Registry import dialog browses OCI registries
The import dialog SHALL provide the same registry browsing flow as the current registry browser (input → repos → tags → manifest → layer).

#### Scenario: Browse registry
- **WHEN** the user enters a registry URL and clicks Browse in the import dialog
- **THEN** the dialog lists repositories from that registry

#### Scenario: Navigate to layer
- **WHEN** the user selects a repo, tag, and inspects a layer
- **THEN** the dialog displays the layer content

### Requirement: Import injects mapping-reference into editor
When the user imports a Gemara artifact layer, the system SHALL parse the imported YAML and inject a `mapping-references` entry into the active editor document.

#### Scenario: Import a Gemara artifact layer
- **WHEN** the user clicks "Import Reference" on a layer containing a valid Gemara artifact
- **THEN** the system extracts `metadata.id`, `metadata.version`, `title`, and `metadata.description` from the imported YAML
- **THEN** the system constructs a `url` from the OCI coordinates (`{registry}/{repo}:{tag}`)
- **THEN** a new entry is appended to the `metadata.mapping-references` list in the editor content
- **THEN** the import dialog closes

#### Scenario: Imported reference shape
- **WHEN** a mapping-reference is injected from an artifact with `metadata.id: SEC.SLAM.CM`, `metadata.version: 1.0.0`, `title: Container Threats`, at OCI reference `ghcr.io/complytime/threats:v1.0.0`
- **THEN** the injected YAML entry is:
  ```yaml
  - id: SEC.SLAM.CM
    title: Container Threats
    version: "1.0.0"
    url: ghcr.io/complytime/threats:v1.0.0
  ```

#### Scenario: Editor has existing mapping-references
- **WHEN** the editor document already has a `mapping-references:` block with entries
- **THEN** the new entry is appended after the existing entries
- **THEN** existing entries are not modified

#### Scenario: Editor has no mapping-references block
- **WHEN** the editor document has a `metadata:` block but no `mapping-references:` key
- **THEN** a `mapping-references:` key is inserted under `metadata:` with the new entry

#### Scenario: Editor has no metadata block
- **WHEN** the editor document has no `metadata:` block
- **THEN** a minimal `metadata:` block with `mapping-references:` is prepended to the document

### Requirement: Import button only appears for Gemara artifacts
The "Import Reference" button SHALL only appear when the inspected layer content is a recognized Gemara artifact.

#### Scenario: Layer is a Gemara artifact
- **WHEN** the layer content matches a known Gemara artifact pattern (has `metadata:` and a recognized top-level key like `threats:`, `controls:`, `guidances:`, etc.)
- **THEN** the "Import Reference" button is displayed alongside "Save to Workspace"

#### Scenario: Layer is not a Gemara artifact
- **WHEN** the layer content does not match any Gemara artifact pattern
- **THEN** only the "Save to Workspace" button is displayed (no "Import Reference")

### Requirement: Import preserves document formatting
The mapping-reference injection SHALL preserve existing YAML formatting, comments, and ordering in the editor document.

#### Scenario: Document with comments
- **WHEN** the editor document contains YAML comments above or within the `metadata:` block
- **THEN** the injected reference does not remove or relocate any comments

#### Scenario: Validation after injection
- **WHEN** a mapping-reference is injected
- **THEN** the resulting document is valid YAML (parseable without errors)
