## REMOVED Requirements

### Requirement: Job creation dialog
**Reason**: Replaced by persistent chat assistant. No discrete job creation needed.
**Migration**: Users interact with the gap analyst via the chat overlay. Context injected from current dashboard state.

### Requirement: Jobs view with active/completed split
**Reason**: Jobs view replaced by dashboard views (Posture, Policies, Evidence, Audit History)
**Migration**: Audit results visible in Audit History view. No job queue UI.

### Requirement: Agent picker
**Reason**: Single agent (gap analyst). No selection needed.
**Migration**: Chat overlay connects directly to the gap analyst agent.
