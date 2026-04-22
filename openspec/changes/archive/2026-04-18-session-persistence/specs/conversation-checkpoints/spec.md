## REMOVED Requirements

### Requirement: Checkpoint button in chat header
**Reason**: Server-side conversation persistence makes manual checkpointing unnecessary. The agent retains context via persistent `taskId` across page refreshes.
**Migration**: No user action needed. Conversation continuity is automatic.

### Requirement: Checkpoint condenses recent turns
**Reason**: Removed with checkpoint feature. Agent context persists natively through the A2A task.
**Migration**: No migration needed.

### Requirement: Checkpoint summary format
**Reason**: Removed with checkpoint feature.
**Migration**: No migration needed.

### Requirement: Checkpoint summary injected on next message
**Reason**: Removed with checkpoint feature. `buildInjectedContext` no longer accepts a checkpoint parameter.
**Migration**: No migration needed. Sticky notes remain for explicit cross-session context.
