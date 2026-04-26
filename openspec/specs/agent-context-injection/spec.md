## ADDED Requirements

### Requirement: Job creation dialog offers artifact selection
The new job dialog SHALL display a list of workspace artifacts that the user can select as input context for the job.

#### Scenario: Workspace has artifacts
- **WHEN** the user opens the new job dialog and the workspace contains artifacts
- **THEN** each workspace artifact is listed with a checkbox
- **THEN** all artifacts are unchecked by default

#### Scenario: Workspace is empty
- **WHEN** the user opens the new job dialog and the workspace is empty
- **THEN** the artifact selection section is hidden
- **THEN** the job creation flow is unchanged from current behavior

#### Scenario: User selects artifacts
- **WHEN** the user checks one or more artifact checkboxes and submits the job
- **THEN** the selected artifact names are recorded on the job as `contextArtifacts`

### Requirement: Selected artifacts are injected into the agent message
The system SHALL serialize selected workspace artifacts into the initial A2A message as additional text parts.

#### Scenario: Single artifact selected
- **WHEN** the job starts with one artifact selected as context
- **THEN** the A2A message includes a text part containing the artifact YAML prefixed with `--- Context: <artifact-name> ---`
- **THEN** the user's prompt text remains the first part of the message

#### Scenario: Multiple artifacts selected
- **WHEN** the job starts with multiple artifacts selected as context
- **THEN** each artifact is included as a separate text part, each with its own `--- Context: <name> ---` header
- **THEN** the artifacts appear after the user's prompt text

#### Scenario: Context size exceeds limit
- **WHEN** the total size of selected artifacts exceeds 100 KB
- **THEN** the system displays a warning before sending
- **THEN** the user can proceed or deselect artifacts to reduce size

### Requirement: Job records context artifact references
The `Job` interface SHALL include a `contextArtifacts` field listing artifact names used as input.

#### Scenario: Job with context artifacts
- **WHEN** a job is created with selected context artifacts
- **THEN** the job's `contextArtifacts` array contains the names of the selected artifacts

#### Scenario: Job without context
- **WHEN** a job is created without selecting any artifacts
- **THEN** the job's `contextArtifacts` is an empty array or undefined

### Requirement: Schema resource preloading limited to lexicon
The agent startup SHALL preload only `gemara://lexicon` (~6K chars) from the Gemara MCP server. The `gemara://schema/definitions` resource (~44K chars) SHALL NOT be preloaded into the system prompt.

#### Scenario: Agent starts with Gemara MCP available
- **WHEN** the agent starts and `GEMARA_MCP_URL` is configured
- **THEN** `_fetch_gemara_resources()` SHALL fetch and inject `gemara://lexicon`
- **THEN** `_fetch_gemara_resources()` SHALL skip `gemara://schema/definitions`

#### Scenario: Agent starts without Gemara MCP
- **WHEN** the agent starts and `GEMARA_MCP_URL` is not configured
- **THEN** no resources are preloaded
- **THEN** the agent operates with prompt and skill knowledge only
