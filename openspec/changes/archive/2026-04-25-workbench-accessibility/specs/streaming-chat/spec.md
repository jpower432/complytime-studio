## MODIFIED Requirements

### Requirement: Streaming text rendering
The chat panel SHALL render partial text events as a "live" message that accumulates tokens. The live message SHALL display a typing cursor while streaming. The streaming message container SHALL be marked with `aria-live="polite"` so screen readers announce new agent responses without interrupting the user.

#### Scenario: Partial text arrives
- **WHEN** an SSE event contains a `TextPart` with metadata `kagent.adk_partial: true`
- **THEN** the chat panel SHALL append the text to the current live agent message
- **THEN** the live message SHALL display a blinking cursor at the end

#### Scenario: Text stream completes
- **WHEN** an SSE event contains a `TextPart` with metadata `kagent.adk_partial: false`
- **THEN** the live message SHALL finalize (cursor removed) and become a permanent message
- **THEN** subsequent partial events SHALL start a new live message

#### Scenario: Screen reader announces completed response
- **WHEN** the agent finishes streaming a response
- **THEN** the finalized message is announced by screen readers via the `aria-live` region
