## MODIFIED Requirements

### Requirement: Unified context builder function
A `buildInjectedContext()` function SHALL assemble all memory sources into a single string for injection into `streamMessage()`. The function SHALL accept dashboard context, sticky notes, and checkpoint summary. Pinned message cache is removed.

#### Scenario: Sticky notes and checkpoint present
- **WHEN** sticky notes and a checkpoint summary exist
- **THEN** the output SHALL be structured as:
  1. `[Dashboard context: {...}]` (existing)
  2. `<sticky-notes>` block with all notes
  3. `<conversation-history>` block with checkpoint summary
  4. User's message text

#### Scenario: Only sticky notes present
- **WHEN** sticky notes exist but no checkpoint summary
- **THEN** the output SHALL include `<sticky-notes>` block and the user's message
- **THEN** no `<conversation-history>` block SHALL be included

#### Scenario: No memory sources
- **WHEN** no sticky notes or checkpoint summary exist
- **THEN** the output SHALL be the dashboard context and user's message only

### Requirement: Context budget enforcement
The `buildInjectedContext()` function SHALL enforce a total character budget of 4500 characters across all injected memory sources.

#### Scenario: Under budget
- **WHEN** total injected context is under 4500 characters
- **THEN** all sources SHALL be included in full

#### Scenario: Over budget
- **WHEN** total injected context exceeds 4500 characters
- **THEN** the checkpoint summary SHALL be truncated
- **THEN** sticky notes SHALL NOT be truncated (already capped at 200 chars each)

### Requirement: New Session clears messages only
The "New Session" button SHALL clear the chat messages array and null the task ID reference. It SHALL NOT save pinned messages or interact with pin cache.

#### Scenario: New Session clicked
- **WHEN** the user clicks "New Session"
- **THEN** the messages array SHALL be cleared
- **THEN** `taskIdRef` SHALL be set to null
- **THEN** no pin cache operations SHALL occur
