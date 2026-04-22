## ADDED Requirements

### Requirement: Chat assistant renders artifact events

The chat assistant SHALL render `TaskArtifactUpdateEvent` payloads in the chat
timeline. When a `TaskArtifactUpdateEvent` includes a part with mimeType
`application/yaml`, the UI SHALL show an artifact card containing the YAML body
and a primary action labeled **Save to Audit History**.

#### Scenario: YAML artifact event received
- **WHEN** a `TaskArtifactUpdateEvent` is received whose part metadata includes
  mimeType `application/yaml`
- **THEN** `chat-assistant.tsx` SHALL render an artifact card with the YAML
  content
- **THEN** the card SHALL include a **Save to Audit History** button

### Requirement: Save artifact to audit history

The chat assistant SHALL persist YAML artifacts to server-side audit history on
user action. When the user activates **Save to Audit History** on an artifact
card, the client SHALL `POST` the YAML payload to `/api/audit-logs`.

#### Scenario: User saves YAML from artifact card
- **WHEN** the user clicks **Save to Audit History** on an artifact card
- **THEN** the chat assistant SHALL send a `POST` with the YAML content to
  `/api/audit-logs`
