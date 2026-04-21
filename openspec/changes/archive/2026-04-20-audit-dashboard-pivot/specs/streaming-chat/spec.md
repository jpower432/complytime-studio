## MODIFIED Requirements

### Requirement: Chat overlay streaming
The chat overlay SHALL render streaming agent responses as they arrive via SSE, displaying text parts incrementally and structured artifacts as complete units when the `TaskArtifactUpdateEvent` is received.

#### Scenario: Streaming text
- **WHEN** the agent streams `TaskStatusUpdateEvent` messages
- **THEN** the chat overlay appends text incrementally to the current message bubble

#### Scenario: Structured artifact received
- **WHEN** the agent emits a `TaskArtifactUpdateEvent` with MIME type `application/yaml`
- **THEN** the chat overlay displays the artifact in a distinct card with a YAML preview and a "Save to Audit History" action button
