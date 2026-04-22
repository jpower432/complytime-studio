## MODIFIED Requirements

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

## REMOVED Requirements

### Requirement: ChatMessage pinned field
**Reason**: Pins were removed in the simple-authz change. This requirement is stale.
**Migration**: No migration needed.

### Requirement: Checkpoint condenses and resets (from lifecycle controls)
**Reason**: Checkpoints are superseded by server-side conversation persistence. The agent retains context via `taskId` across refreshes. Manual condensation is unnecessary.
**Migration**: Users rely on natural conversation continuity. Use sticky notes for persistent cross-session context.
