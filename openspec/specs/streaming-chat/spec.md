## Requirements

### Requirement: Enable kagent streaming

All Agent CRDs SHALL have `stream: true` set in the `declarative` block of `agent-specialists.yaml`. This enables the ADK runner to emit partial text events and tool call DataParts via SSE.

#### Scenario: Agent CRDs include stream flag
- **WHEN** the Helm chart is rendered with `helm template`
- **THEN** every Agent CRD with a `declarative` block SHALL include `stream: true`

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

The chat panel SHALL display session management actions in the header: "New Session" and a sticky notes toggle button. The "New Session" button resets the A2A task and clears server-side conversation state.

#### Scenario: New Session resets with server clear
- **WHEN** the user clicks "New Session"
- **THEN** the client SHALL PUT empty state to `/api/chat/history`
- **THEN** the `messages` array SHALL be cleared
- **THEN** the `taskIdRef` SHALL be set to null

#### Scenario: Sticky notes toggle
- **WHEN** the user clicks the sticky notes button
- **THEN** the sticky notes panel SHALL toggle open/closed

### Requirement: Save AuditLog from chat
The `saveAuditLog` function SHALL send only `policy_id` and `content` to `POST /api/audit-logs`. It SHALL NOT send `audit_start`, `audit_end`, or `summary` — the gateway derives these from the YAML content.

#### Scenario: User clicks Save to Audit History
- **WHEN** the user clicks "Save to Audit History" on an artifact card
- **THEN** `saveAuditLog` SHALL POST `{"policy_id": selectedPolicyId, "content": artifact.content}` to `/api/audit-logs`

#### Scenario: Gateway returns parse error
- **WHEN** the gateway returns `400 Bad Request` because the artifact content is invalid AuditLog YAML
- **THEN** the UI SHALL display the error message to the user

### Requirement: Artifact save button is a confirmation action

Previously: The "Save to Audit History" button was the **only** path to persist an agent-produced artifact.

The "Save to Audit History" button SHALL remain available on artifact cards for admin users. When server-side auto-persistence is enabled, the button SHALL function as an idempotent confirmation or re-save action. The artifact card SHALL display an "Auto-saved" indicator when the artifact was persisted server-side.

#### Scenario: Auto-persist enabled, artifact displayed
- **WHEN** an artifact card is rendered and `AUTO_PERSIST_ARTIFACTS` is enabled
- **THEN** the card SHALL display an "Auto-saved" text indicator
- **AND** the "Save to Audit History" button SHALL still be available for admin users

#### Scenario: Auto-persist disabled, artifact displayed
- **WHEN** an artifact card is rendered and `AUTO_PERSIST_ARTIFACTS` is disabled
- **THEN** the card SHALL NOT display an "Auto-saved" indicator
- **AND** the "Save to Audit History" button SHALL be the primary save action (current behavior)

#### Scenario: Manual save after auto-persist
- **WHEN** an admin clicks "Save to Audit History" on an auto-persisted artifact
- **THEN** the `POST /api/audit-logs` request SHALL succeed
- **AND** `ReplacingMergeTree` SHALL deduplicate the row if content is unchanged

### Requirement: Agent context reflects populated signals
The dashboard context injected into agent messages SHALL include non-null values for all shared signals: `policy_id`, `time_range_start`, `time_range_end`, `control_id`, `requirement_id`, `eval_result`.

#### Scenario: Full context after cross-view navigation
- **WHEN** the user has navigated posture -> requirements (setting policy and time range) and opens chat
- **THEN** the injected context JSON includes `policy_id`, `time_range_start`, and `time_range_end` with the values from the shared signals

### Requirement: Chat pre-loads active policy context
The chat assistant SHALL pre-load the active policy context when the user opens the chat from a policy detail view. The first message SHALL include the policy ID, active tab, and any selected filters as context for the agent.

#### Scenario: Chat opened from posture detail
- **WHEN** the user opens the chat FAB while viewing policy "ampel-branch-protection" on the Requirements tab
- **THEN** the chat sends the first user message with context: `{"policy_id":"ampel-branch-protection","view":"requirements","filters":{...}}`

#### Scenario: Chat opened from posture grid
- **WHEN** the user opens the chat FAB from the top-level posture view (no policy selected)
- **THEN** the chat sends no pre-loaded context and the agent asks which policy to work with
