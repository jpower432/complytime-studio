## ADDED Requirements

### Requirement: Consecutive agent messages grouped into single block

The chat panel SHALL group consecutive messages from the same role into a single visual block. A new block starts only when the role changes.

#### Scenario: Agent text followed by tool call followed by agent text
- **WHEN** the message array contains [agent-text, tool-call, agent-text] with no intervening user messages
- **THEN** the chat panel SHALL render one agent block containing text, an inline tool call, and more text

#### Scenario: User message breaks grouping
- **WHEN** the message array contains [agent-text, user-text, agent-text]
- **THEN** the chat panel SHALL render three separate blocks (agent, user, agent)

### Requirement: Tool calls render inline within agent blocks

Tool call blocks SHALL appear inline within their parent agent message group, not as standalone message bubbles.

#### Scenario: Completed tool call in agent block
- **WHEN** a tool call appears between agent text segments
- **THEN** it SHALL render as a collapsed inline block showing tool name and status icon
- **THEN** the user SHALL be able to expand it to see result details

#### Scenario: Pending approval tool call in agent block
- **WHEN** a tool call with `is_long_running: true` appears in an agent block
- **THEN** it SHALL render inline with Approve/Reject buttons visible without expanding
