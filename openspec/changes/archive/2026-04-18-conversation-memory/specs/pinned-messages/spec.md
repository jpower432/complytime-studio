## ADDED Requirements

### Requirement: Pin button on agent messages

Each agent message in the chat panel SHALL display a pin toggle button. User messages SHALL NOT have pin buttons.

#### Scenario: Agent message displays pin toggle
- **WHEN** an agent message is rendered in the chat panel
- **THEN** a pin toggle button SHALL appear on hover or focus
- **THEN** the button SHALL indicate the current pinned state (pinned vs unpinned)

#### Scenario: User message has no pin toggle
- **WHEN** a user message is rendered in the chat panel
- **THEN** no pin toggle button SHALL be displayed

### Requirement: Toggle pin state on click

Clicking the pin button SHALL toggle the `pinned` boolean on the `ChatMessage`. The `ChatMessage` interface SHALL include an optional `pinned` field.

#### Scenario: Pin an unpinned message
- **WHEN** the user clicks the pin button on an unpinned agent message
- **THEN** the message's `pinned` field SHALL be set to `true`
- **THEN** the message SHALL display a visual pinned indicator

#### Scenario: Unpin a pinned message
- **WHEN** the user clicks the pin button on a pinned agent message
- **THEN** the message's `pinned` field SHALL be set to `false`
- **THEN** the visual pinned indicator SHALL be removed

### Requirement: Pin limit enforcement

The system SHALL enforce a maximum of 5 pinned messages at any time.

#### Scenario: Pin limit reached
- **WHEN** the user pins a 6th message
- **THEN** the oldest pinned message (by array position) SHALL be automatically unpinned
- **THEN** a brief notification SHALL inform the user which message was unpinned

#### Scenario: Under pin limit
- **WHEN** fewer than 5 messages are pinned
- **THEN** pinning a new message SHALL succeed without affecting existing pins

### Requirement: Pinned messages persist in localStorage

Pinned state SHALL be saved as part of the existing `studio-chat-history` localStorage entry via the `pinned` field on `ChatMessage`.

#### Scenario: Pin state survives page reload
- **WHEN** the user pins a message and reloads the page
- **THEN** the message SHALL still display as pinned after reload

### Requirement: Pinned messages carry across session reset

On "New Session," pinned messages SHALL be written to `studio-pinned-cache` in localStorage. The cache SHALL be read and injected as `<conversation-history>` on the next `streamMessage()` call, then cleared.

#### Scenario: New Session with pinned messages
- **WHEN** the user clicks "New Session" and 2 messages are pinned
- **THEN** those 2 messages SHALL be serialized to `studio-pinned-cache`
- **THEN** the chat UI messages SHALL be cleared
- **THEN** the `taskIdRef` SHALL be set to null

#### Scenario: First message after session reset
- **WHEN** the user sends the first message of a new session and `studio-pinned-cache` contains entries
- **THEN** the cached pinned messages SHALL be serialized as a `<conversation-history>` block in the `streamMessage()` text
- **THEN** each pinned message SHALL be truncated to 500 characters
- **THEN** `studio-pinned-cache` SHALL be cleared after injection
