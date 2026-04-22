## ADDED Requirements

### Requirement: Unified context builder function

A `buildInjectedContext()` function SHALL assemble all memory sources into a single string for injection into `streamMessage()`. The function SHALL accept dashboard context and sticky notes only.

#### Scenario: Sticky notes present
- **WHEN** sticky notes exist
- **THEN** the output SHALL be structured as:
  1. `[Dashboard context: {...}]` (existing)
  2. `<sticky-notes>` block with all notes
  3. User's message text

#### Scenario: No memory sources
- **WHEN** no sticky notes exist
- **THEN** the output SHALL be the dashboard context and user's message only

### Requirement: Context budget enforcement

The `buildInjectedContext()` function SHALL enforce a total character budget of 4500 characters across all injected memory sources.

#### Scenario: Under budget
- **WHEN** total injected context is under 4500 characters
- **THEN** all sources SHALL be included in full

#### Scenario: Over budget
- **WHEN** total injected context exceeds 4500 characters
- **THEN** sticky notes SHALL be truncated from oldest first

### Requirement: New Session clears messages only

The "New Session" button SHALL clear the chat messages array, null the task ID reference, and clear server-side state via `PUT /api/chat/history`.

#### Scenario: New Session clicked
- **WHEN** the user clicks "New Session"
- **THEN** the messages array SHALL be cleared
- **THEN** `taskIdRef` SHALL be set to null
- **THEN** server-side state SHALL be cleared

### Requirement: Context injection indicator

On the first message of a new session, the UI SHALL display a collapsed "Context sent" block showing what was injected.

#### Scenario: Context was injected
- **WHEN** the first message of a session includes injected context (sticky notes)
- **THEN** a collapsed block SHALL appear above the first user message
- **THEN** expanding the block SHALL show the exact text that was injected
- **THEN** the block SHALL be labeled "Memory context sent to agent"

#### Scenario: No context injected
- **WHEN** the first message has no injected context
- **THEN** no context indicator block SHALL appear
