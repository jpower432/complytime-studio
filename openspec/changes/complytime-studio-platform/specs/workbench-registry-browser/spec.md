## ADDED Requirements

### Requirement: Registry browser view in workbench
The workbench SHALL provide a registry browser view accessible from the sidebar navigation, allowing users to discover and inspect OCI bundles.

#### Scenario: User navigates to registry browser
- **WHEN** the user selects "Registry" in the sidebar
- **THEN** the workbench SHALL display a registry browser view with a search/input field for registry references

### Requirement: Repository and tag listing
The workbench SHALL list repositories and tags from OCI registries via the orchestrator's oras-mcp tools.

#### Scenario: User enters a registry URL
- **WHEN** the user enters a registry URL (e.g., `registry.io/org`)
- **THEN** the workbench SHALL display available repositories

#### Scenario: User selects a repository
- **WHEN** the user selects a repository
- **THEN** the workbench SHALL list available tags with version identifiers

### Requirement: Bundle layer inspection
The workbench SHALL display the layers of a selected bundle with their media types and artifact names.

#### Scenario: User selects a tag
- **WHEN** the user selects a specific tag
- **THEN** the workbench SHALL fetch the manifest and display each layer's media type and inferred artifact name

### Requirement: Pull artifact into editor
The workbench SHALL allow users to pull individual artifact layers from a registry bundle and load them into the YAML editor.

#### Scenario: User clicks Load on a layer
- **WHEN** the user clicks "Load in Editor" on a specific layer
- **THEN** the workbench SHALL fetch the layer content and open it as a new tab in the artifact editor panel

#### Scenario: Pull failure
- **WHEN** fetching a layer fails (auth error, network error)
- **THEN** the workbench SHALL display an error message to the user

### Requirement: Registry browser uses oras-mcp proxy
The workbench SHALL communicate with oras-mcp through the orchestrator's backend proxy, not directly to the MCP server.

#### Scenario: API routing
- **WHEN** the registry browser makes a request
- **THEN** the request SHALL go through a `/api/registry/*` proxy endpoint on the orchestrator that delegates to oras-mcp tools
