## ADDED Requirements

### Requirement: BYO ADK agent container
The system SHALL deploy the gap analyst as a standalone Python container using Google ADK `LlmAgent` wrapped in `A2aAgentExecutor`, with gemara-mcp and clickhouse-mcp as sidecar containers using stdio transport.

#### Scenario: Agent pod starts
- **WHEN** the Helm chart is deployed with the BYO gap analyst enabled
- **THEN** the pod starts with three containers (agent, gemara-mcp sidecar, clickhouse-mcp sidecar) and the agent exposes an A2A endpoint on port 8080

### Requirement: Deterministic input gate
The agent SHALL run a `before_agent_callback` that parses the user message, validates required inputs (policy reference, audit timeline), pre-queries ClickHouse for target inventory and evidence summary, and loads MCP resources (`gemara://lexicon`, `gemara://schema/definitions`).

#### Scenario: Missing policy reference
- **WHEN** the user sends a message without a policy reference or audit timeline
- **THEN** the `before_agent_callback` returns a structured error message asking for the missing inputs without invoking the LLM

#### Scenario: Successful pre-processing
- **WHEN** the user provides a valid policy reference and audit timeline
- **THEN** the callback queries ClickHouse for target inventory and evidence counts, loads MCP resources, and injects all structured context into the agent's system prompt

### Requirement: Deterministic output gate
The agent SHALL run an `after_agent_callback` that extracts YAML from the agent output, validates it via gemara-mcp (`#AuditLog`), checks completeness (every criteria entry has a corresponding AuditResult), and calls `save_artifact` on success.

#### Scenario: Valid AuditLog output
- **WHEN** the agent produces a valid, complete AuditLog
- **THEN** the callback validates it, saves it as an artifact with MIME type `application/yaml`, and the event converter emits a `TaskArtifactUpdateEvent`

#### Scenario: Validation failure with retry
- **WHEN** the agent output fails gemara-mcp validation
- **THEN** the callback returns the validation errors to the agent for retry (max 3 attempts)

### Requirement: Structured artifact emission
The agent SHALL use a custom event converter that inspects `artifact_delta` on ADK events and emits `TaskArtifactUpdateEvent` with typed parts (MIME type `application/yaml`, artifact filename).

#### Scenario: Client receives structured artifact
- **WHEN** the agent saves a validated AuditLog artifact
- **THEN** the A2A stream includes a `TaskArtifactUpdateEvent` with a part containing the YAML content and metadata (mimeType, name)

### Requirement: MCP resource access
The agent SHALL instantiate `McpToolset` with `use_mcp_resources=True` for the gemara-mcp sidecar, enabling the LLM to call `load_mcp_resource` for `gemara://lexicon` and `gemara://schema/definitions`.

#### Scenario: Agent reads lexicon
- **WHEN** the LLM calls `load_mcp_resource` with URI `gemara://lexicon`
- **THEN** the MCP toolset fetches the resource from the gemara-mcp sidecar and returns the content to the LLM
