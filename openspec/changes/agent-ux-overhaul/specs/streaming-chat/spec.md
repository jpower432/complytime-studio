## MODIFIED Requirements

### Requirement: Streaming text rendering

The chat panel SHALL render partial text events as a "live" message that accumulates tokens. The live message SHALL display a typing cursor while streaming. When streaming finalizes, the message SHALL remain part of the current agent block rather than creating a new bubble.

#### Scenario: Partial text arrives
- **WHEN** an SSE event contains a `TextPart` with metadata `kagent.adk_partial: true`
- **THEN** the chat panel SHALL append the text to the current live agent message within the active agent block

#### Scenario: Text stream completes
- **WHEN** an SSE event contains a `TextPart` with metadata `kagent.adk_partial: false`
- **THEN** the live message SHALL finalize (cursor removed) within the current agent block
- **THEN** subsequent partial events SHALL continue in the same agent block (not create a new bubble) unless a user message intervenes
