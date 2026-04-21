## ADDED Requirements

### Requirement: Conversation history replay on follow-up messages

The workbench SHALL assemble the full conversation history and re-send it with every follow-up A2A message so agents retain context across turns in a stateless runtime.

#### Scenario: Follow-up reply includes history
- **WHEN** the user sends a follow-up reply on an existing job
- **THEN** `streamReply` SHALL include all prior messages (user and agent) serialized in a `<conversation-history>` block as the first message part
- **THEN** each message in the block SHALL be prefixed with `[User]:` or `[Agent]:`

#### Scenario: Context artifacts re-sent on follow-up
- **WHEN** the job has context artifacts selected at creation
- **THEN** every follow-up reply SHALL include those artifacts as additional message parts with `--- Context: <name> ---` delimiters

### Requirement: Token budget with oldest-first truncation

The workbench SHALL enforce a character budget on replayed conversation history to prevent unbounded token usage.

#### Scenario: History within budget
- **WHEN** the serialized conversation history is under 100,000 characters
- **THEN** the full history SHALL be included without modification

#### Scenario: History exceeds budget
- **WHEN** the serialized conversation history exceeds 100,000 characters
- **THEN** the oldest messages SHALL be removed until the total is within budget
- **THEN** context artifacts and the most recent 4 messages SHALL always be preserved
- **THEN** the history block SHALL begin with `[Earlier conversation truncated]`

### Requirement: Input sanitization on replayed content

Replayed conversation history and context artifacts SHALL be treated as untrusted data with structural guardrails to prevent prompt injection.

#### Scenario: History wrapped in delimiters
- **WHEN** conversation history is assembled for replay
- **THEN** it SHALL be enclosed in `<conversation-history>` and `</conversation-history>` tags

#### Scenario: Artifacts labeled as reference
- **WHEN** context artifacts are included in the replay
- **THEN** each artifact SHALL be prefixed with `--- Context: <name> (reference only) ---`
