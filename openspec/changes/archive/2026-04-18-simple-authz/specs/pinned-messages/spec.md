## REMOVED Requirements

### Requirement: Pin button on agent messages
**Reason**: Pins are consumable (inject once, then vanish) which confuses users. Sticky notes provide persistent memory. Checkpoints provide session-boundary summarization. Pins duplicate checkpoint behavior with worse UX.
**Migration**: Users should use sticky notes for persistent context and checkpoints for session transitions.

### Requirement: Toggle pin state on click
**Reason**: Removed with pin feature.
**Migration**: No migration needed.

### Requirement: Pin limit enforcement
**Reason**: Removed with pin feature.
**Migration**: No migration needed.

### Requirement: Pinned messages persist in localStorage
**Reason**: Removed with pin feature.
**Migration**: `studio-pinned-cache` localStorage key can be cleaned up by the application on next load.

### Requirement: Pinned messages carry across session reset
**Reason**: Removed with pin feature. Checkpoints serve the same purpose more transparently.
**Migration**: Use the Checkpoint button to condense context before starting a new session.
