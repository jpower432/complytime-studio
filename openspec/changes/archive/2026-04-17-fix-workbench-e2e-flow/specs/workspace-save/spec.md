## ADDED Requirements

### Requirement: Gateway provides workspace save endpoint
The gateway SHALL expose `POST /api/workspace/save` that writes artifact content to the `.complytime/artifacts/` directory within the workspace root.

#### Scenario: Successful save
- **WHEN** a POST request is sent with `{"filename": "threat-catalog.yaml", "content": "metadata:\n..."}`
- **THEN** the gateway writes the content to `.complytime/artifacts/threat-catalog.yaml`
- **THEN** the response is `{"path": ".complytime/artifacts/threat-catalog.yaml"}`

#### Scenario: Path traversal rejected
- **WHEN** a POST request is sent with `{"filename": "../../../etc/passwd", "content": "..."}`
- **THEN** the gateway returns HTTP 400 with `{"error": "invalid filename"}`
- **THEN** no file is written

#### Scenario: Directory auto-created
- **WHEN** a POST request is sent and `.complytime/artifacts/` does not yet exist
- **THEN** the gateway creates the directory before writing the file

#### Scenario: File overwrite
- **WHEN** a POST request is sent with a filename that already exists in `.complytime/artifacts/`
- **THEN** the existing file is overwritten with the new content

### Requirement: Registry browser layer view has save action
The registry browser layer view SHALL include a "Save to Workspace" button that saves the displayed layer content to the local workspace.

#### Scenario: Save layer content
- **WHEN** user clicks "Save to Workspace" on a layer view
- **THEN** the system calls `POST /api/workspace/save` with the layer content and a filename derived from the repository name and media type
- **THEN** a success message displays the saved file path

#### Scenario: Save fails
- **WHEN** user clicks "Save to Workspace" and the save endpoint returns an error
- **THEN** the error message is displayed in the registry browser

### Requirement: Artifact panel has save action
The artifact panel SHALL include a "Save" button that writes the current artifact content to the local workspace.

#### Scenario: Save mission artifact
- **WHEN** user clicks "Save" in the artifact panel toolbar
- **THEN** the system calls `POST /api/workspace/save` with the artifact YAML and the artifact name as filename
- **THEN** a brief success indicator appears

### Requirement: Registry browser returns valid JSON on errors
The gateway registry proxy SHALL return valid JSON error responses when the underlying MCP tool returns non-JSON content.

#### Scenario: MCP tool returns text error
- **WHEN** an oras-mcp tool returns plain text (e.g., "invalid registry")
- **THEN** the gateway responds with HTTP 502 and `{"error": "invalid registry"}`
- **THEN** the response Content-Type is `application/json`

### Requirement: Registry browser layer endpoint calls correct tool
The `/api/registry/layer` endpoint SHALL call the `fetch_layer` MCP tool.

#### Scenario: Fetch layer content
- **WHEN** a GET request is made to `/api/registry/layer?ref=ghcr.io/org/repo@sha256:abc`
- **THEN** the gateway calls the `fetch_layer` oras-mcp tool with the provided reference
- **THEN** the layer content is returned to the client
