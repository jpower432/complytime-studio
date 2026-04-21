## ADDED Requirements

### Requirement: Persistent chat icon
The system SHALL display a chat icon in the bottom-right corner of every dashboard view that opens a chat overlay window when clicked.

#### Scenario: Open chat
- **WHEN** the user clicks the chat icon from any dashboard view
- **THEN** a chat overlay window appears anchored to the bottom-right corner with a message input and conversation history

#### Scenario: Close chat
- **WHEN** the user clicks the close button or the chat icon again
- **THEN** the chat overlay closes and the conversation state is preserved

### Requirement: Dashboard context injection
The system SHALL automatically inject the user's current dashboard context (active policy, selected time range, selected framework) into the chat message as metadata.

#### Scenario: Chat from Posture view
- **WHEN** the user opens chat while viewing the Posture dashboard with a specific policy selected
- **THEN** the system includes the policy_id and current audit period in the A2A message context

#### Scenario: Chat from Evidence view
- **WHEN** the user opens chat while viewing filtered evidence
- **THEN** the system includes the active filter parameters (policy_id, target_id, time range) in the A2A message context

### Requirement: Conversation persistence
The system SHALL persist the chat conversation in browser storage across page navigations within the same session.

#### Scenario: Navigate between views
- **WHEN** the user navigates from Posture to Evidence view with an active chat conversation
- **THEN** the chat overlay retains the full conversation history

### Requirement: AuditLog artifact handling
The system SHALL detect structured `TaskArtifactUpdateEvent` from the agent and display AuditLog artifacts with a "Save to Audit History" action.

#### Scenario: Agent produces AuditLog
- **WHEN** the agent emits a `TaskArtifactUpdateEvent` with MIME type `application/yaml`
- **THEN** the chat displays the artifact with a preview and a "Save to Audit History" button that stores it in ClickHouse
