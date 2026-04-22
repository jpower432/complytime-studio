## ADDED Requirements

### Requirement: Checkpoint button in chat header

The chat panel header SHALL display a "Checkpoint" button alongside "New Session" and the sticky notes toggle.

#### Scenario: Checkpoint button visible during conversation
- **WHEN** the chat panel is open and at least one exchange (user + agent) exists
- **THEN** a "Checkpoint" button SHALL be visible in the header

#### Scenario: Checkpoint button hidden when empty
- **WHEN** the chat panel is open and no messages exist
- **THEN** the "Checkpoint" button SHALL be disabled or hidden

### Requirement: Checkpoint condenses recent turns

Clicking "Checkpoint" SHALL serialize messages since the last checkpoint (or conversation start) into a summary string, reset the A2A `taskId`, and insert a visual divider in the message list.

#### Scenario: First checkpoint in a conversation
- **WHEN** the user clicks "Checkpoint" after 8 turns of conversation
- **THEN** the 8 turns SHALL be condensed into a summary string
- **THEN** a visual checkpoint divider SHALL appear in the message list
- **THEN** the `taskIdRef` SHALL be set to null
- **THEN** messages above the divider SHALL remain visible (read-only history)

#### Scenario: Second checkpoint
- **WHEN** the user clicks "Checkpoint" after 5 more turns following a previous checkpoint
- **THEN** only the 5 turns since the last checkpoint SHALL be condensed
- **THEN** a new visual checkpoint divider SHALL be inserted

### Requirement: Checkpoint summary format

The checkpoint summary SHALL use a deterministic, naive serialization: each turn condensed to `"User: <first 100 chars> → Agent: <first 200 chars>"`, joined by newlines.

#### Scenario: Summary generation
- **WHEN** a checkpoint is triggered with 4 turns since last checkpoint
- **THEN** the summary SHALL contain 4 lines, one per turn pair
- **THEN** user text SHALL be truncated at 100 characters with ellipsis
- **THEN** agent text SHALL be truncated at 200 characters with ellipsis

### Requirement: Checkpoint summary injected on next message

The checkpoint summary SHALL be injected as part of `<conversation-history>` on the next `streamMessage()` call, combined with any pinned message cache.

#### Scenario: First message after checkpoint
- **WHEN** the user sends a message after a checkpoint
- **THEN** `streamMessage()` SHALL be called (new task, no taskId)
- **THEN** the checkpoint summary SHALL appear in the `<conversation-history>` block
- **THEN** subsequent messages SHALL use `streamReply()` with the new taskId
