## MODIFIED Requirements

### Requirement: Lifecycle controls in chat panel

The chat panel SHALL display session management actions in the header: "New Session" (replaces "Clear"), "Checkpoint", and a sticky notes toggle button. The "New Session" button resets the A2A task while carrying forward pinned messages and sticky notes. The "Checkpoint" button condenses recent turns and resets the task mid-conversation.

#### Scenario: New Session resets with carry-forward
- **WHEN** the user clicks "New Session"
- **THEN** pinned messages SHALL be written to `studio-pinned-cache` in localStorage
- **THEN** the `messages` array SHALL be cleared
- **THEN** the `taskIdRef` SHALL be set to null
- **THEN** the next `streamMessage()` call SHALL include injected context from pins and sticky notes

#### Scenario: Checkpoint condenses and resets
- **WHEN** the user clicks "Checkpoint"
- **THEN** messages since last checkpoint SHALL be serialized into a summary
- **THEN** a visual divider SHALL be inserted in the message list
- **THEN** the `taskIdRef` SHALL be set to null
- **THEN** messages above the divider SHALL remain visible

#### Scenario: Sticky notes toggle
- **WHEN** the user clicks the sticky notes button
- **THEN** the sticky notes panel SHALL toggle open/closed

## ADDED Requirements

### Requirement: ChatMessage pinned field

The `ChatMessage` interface SHALL include an optional `pinned: boolean` field. The `saveHistory` function SHALL persist the `pinned` field alongside existing fields.

#### Scenario: Pinned field persisted
- **WHEN** a message has `pinned: true` and `saveHistory` runs
- **THEN** the `pinned` field SHALL be present in the serialized JSON in localStorage
