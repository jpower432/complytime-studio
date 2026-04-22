## ADDED Requirements

### Requirement: Custom ADK-to-A2A event converter

The agent SHALL configure `A2aAgentExecutor` with a custom `adk_event_converter`
callable on `A2aAgentExecutorConfig`. The converter SHALL map ADK events to A2A
events: when an ADK event carries `artifact_delta`, emit a
`TaskArtifactUpdateEvent` whose parts include `application/yaml` mimeType
metadata; when the ADK event contains only text content, emit a standard
`TaskStatusUpdateEvent`.

#### Scenario: ADK event includes artifact_delta
- **WHEN** an ADK event includes `artifact_delta`
- **THEN** the converter SHALL emit a `TaskArtifactUpdateEvent`
- **THEN** emitted parts SHALL declare `application/yaml` mimeType metadata

#### Scenario: ADK event is text-only
- **WHEN** an ADK event has text content only (no `artifact_delta`)
- **THEN** the converter SHALL emit a standard `TaskStatusUpdateEvent`

### Requirement: Manual A2aAgentExecutor with AgentCardBuilder

The application SHALL construct `A2aAgentExecutor` manually (not via `to_a2a`)
at process start. The executor SHALL be created with `A2aAgentExecutorConfig`
including the custom `adk_event_converter`. The agent card SHALL be
auto-generated using `AgentCardBuilder` (or equivalent supported builder API).

#### Scenario: Agent process starts
- **WHEN** the agent process starts
- **THEN** it SHALL construct `A2aAgentExecutor` with `A2aAgentExecutorConfig`
  containing the custom `adk_event_converter`
- **THEN** the agent card SHALL be auto-generated via `AgentCardBuilder`
