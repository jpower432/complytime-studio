## ADDED Requirements

### Requirement: Agent recognizes sticky-notes context tag

The agent prompt SHALL document the `<sticky-notes>` tag convention. Content within `<sticky-notes>` tags represents persistent user-curated facts. The agent SHALL treat these as always-true background context unless explicitly contradicted by the user.

#### Scenario: Sticky notes present in message
- **WHEN** a user message contains a `<sticky-notes>` block
- **THEN** the agent SHALL treat each note as a persistent fact for the duration of the conversation
- **THEN** the agent SHALL NOT ask the user to re-confirm information already in sticky notes

#### Scenario: User contradicts a sticky note
- **WHEN** the user provides information that contradicts a sticky note
- **THEN** the agent SHALL use the user's latest statement and note the discrepancy

### Requirement: Agent suggests sticky notes for persistent facts

The agent prompt SHALL instruct the assistant to suggest saving persistent facts as sticky notes when the user establishes scope, dates, priorities, or recurring parameters.

#### Scenario: User establishes audit window
- **WHEN** the user states "our audit window is Q1 2026" or equivalent scope-setting fact
- **THEN** the agent SHALL include a suggestion: "Tip: save 'Audit window: Q1 2026' as a sticky note to carry this across sessions."

#### Scenario: Agent does not auto-create
- **WHEN** the agent suggests a sticky note
- **THEN** the agent SHALL NOT create the note automatically
- **THEN** the user SHALL manually add the note via the sticky notes panel
