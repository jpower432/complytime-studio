## ADDED Requirements

### Requirement: Enable kagent streaming

All Agent CRDs SHALL have `stream: true` set in the `declarative` block of `agent-specialists.yaml`. This enables the ADK runner to emit partial text events and tool call DataParts via SSE.

#### Scenario: Agent CRDs include stream flag
- **WHEN** the Helm chart is rendered with `helm template`
- **THEN** every Agent CRD with a `declarative` block SHALL include `stream: true`

### Requirement: Streaming text rendering

The chat panel SHALL render partial text events as a "live" message that accumulates tokens. The live message SHALL display a typing cursor while streaming.

#### Scenario: Partial text arrives
- **WHEN** an SSE event contains a `TextPart` with metadata `kagent.adk_partial: true`
- **THEN** the chat panel SHALL append the text to the current live agent message
- **THEN** the live message SHALL display a blinking cursor at the end

#### Scenario: Text stream completes
- **WHEN** an SSE event contains a `TextPart` with metadata `kagent.adk_partial: false`
- **THEN** the live message SHALL finalize (cursor removed) and become a permanent message
- **THEN** subsequent partial events SHALL start a new live message

### Requirement: Tool call blocks

The chat panel SHALL render `function_call` DataParts as collapsible blocks showing the tool name. Tool results SHALL update the corresponding block.

#### Scenario: Tool call emitted
- **WHEN** an SSE event contains a `DataPart` with metadata type `function_call`
- **THEN** the chat panel SHALL render a tool call block showing the tool name
- **THEN** the block SHALL be in an "executing" state with a spinner

#### Scenario: Tool result received
- **WHEN** an SSE event contains a `DataPart` with metadata type `function_response` matching a previous tool call
- **THEN** the corresponding tool call block SHALL update to show a summary of the result
- **THEN** the block SHALL collapse automatically
- **THEN** the user SHALL be able to expand the block to see full result details

### Requirement: HITL approve/reject for requireApproval tools

The chat panel SHALL render approve and reject buttons on tool call blocks that have the `is_long_running: true` metadata flag. The agent SHALL pause in `input-required` state until the user acts.

#### Scenario: Tool requires approval
- **WHEN** an SSE event contains a `function_call` DataPart with `kagent.is_long_running: true`
- **THEN** the tool call block SHALL display "Approve" and "Reject" buttons
- **THEN** the block SHALL display "Waiting for your approval" label
- **THEN** the reply input SHALL remain disabled until the user approves or rejects

#### Scenario: User approves tool call
- **WHEN** the user clicks "Approve" on a tool call block
- **THEN** the system SHALL send an A2A message with the approval decision
- **THEN** the tool call block SHALL transition to "executing" state
- **THEN** the agent SHALL resume execution

#### Scenario: User rejects tool call
- **WHEN** the user clicks "Reject" on a tool call block
- **THEN** the system SHALL send an A2A message with the rejection decision
- **THEN** the tool call block SHALL display "Rejected" status
- **THEN** the agent SHALL resume with the rejection context

### Requirement: Lifecycle controls in chat panel

The chat panel SHALL display job lifecycle actions (Cancel, Accept) at the bottom of the panel, appropriate to the current job status.

#### Scenario: Working job shows cancel
- **WHEN** a job has status `working`
- **THEN** the chat panel SHALL display a "Cancel Job" button

#### Scenario: Ready job shows accept and cancel
- **WHEN** a job has status `ready`
- **THEN** the chat panel SHALL display "Accept" and "Cancel Job" buttons
- **THEN** the "Accept" button SHALL open the acceptance note dialog

#### Scenario: Completed states hide controls
- **WHEN** a job has status `accepted`, `cancelled`, or `failed`
- **THEN** the chat panel SHALL not display lifecycle action buttons
