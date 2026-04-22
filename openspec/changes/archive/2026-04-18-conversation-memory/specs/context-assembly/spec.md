## ADDED Requirements

### Requirement: Unified context builder function

A `buildInjectedContext()` function SHALL assemble all memory sources into a single string for injection into `streamMessage()`.

#### Scenario: All sources present
- **WHEN** sticky notes, pinned cache, and a checkpoint summary all exist
- **THEN** the output SHALL be structured as:
  1. `[Dashboard context: {...}]` (existing)
  2. `<sticky-notes>` block with all notes
  3. `<conversation-history>` block with pinned messages and checkpoint summary
  4. User's message text

#### Scenario: Only sticky notes present
- **WHEN** sticky notes exist but no pinned cache or checkpoint summary
- **THEN** the output SHALL include `<sticky-notes>` block and the user's message
- **THEN** no `<conversation-history>` block SHALL be included

#### Scenario: No memory sources
- **WHEN** no sticky notes, pinned cache, or checkpoint summary exist
- **THEN** the output SHALL be the dashboard context and user's message only (current behavior)

### Requirement: Context budget enforcement

The `buildInjectedContext()` function SHALL enforce a total character budget of 4500 characters across all injected memory sources.

#### Scenario: Under budget
- **WHEN** total injected context is under 4500 characters
- **THEN** all sources SHALL be included in full (subject to per-item truncation)

#### Scenario: Over budget
- **WHEN** total injected context exceeds 4500 characters
- **THEN** pinned messages SHALL be truncated further (shortest first)
- **THEN** sticky notes SHALL NOT be truncated (already capped at 200 chars each)

### Requirement: Context injection indicator

On the first message of a new session, the UI SHALL display a collapsed "Context sent" block showing what was injected.

#### Scenario: Context was injected
- **WHEN** the first message of a session includes injected context (sticky notes, pins, or checkpoint)
- **THEN** a collapsed block SHALL appear above the first user message
- **THEN** expanding the block SHALL show the exact text that was injected
- **THEN** the block SHALL be labeled "Memory context sent to agent"

#### Scenario: No context injected
- **WHEN** the first message has no injected context
- **THEN** no context indicator block SHALL appear
