## Why

The chat assistant loses all conversation context on page refresh, tab close, or pod restart. Users report the agent "forgets what I typed earlier" during normal workflows. Client-side workarounds (checkpoints, sticky notes) require manual action and don't survive cross-device or cross-tab usage. See `decisions/001-session-persistence-storage.md`.

## What Changes

- Add server-side conversation turn storage in the gateway, using the same `MemorySessionStore` pattern as auth sessions
- Add REST endpoints for saving and loading conversation turns (`GET/PUT /api/chat/history`)
- Modify the workbench chat panel to persist turns to the server on each exchange and hydrate on load
- Remove checkpoint feature (redundant — server-side persistence provides continuity without manual intervention)
- Update `streaming-chat` spec to reflect removal of checkpoint lifecycle controls
- Update `context-assembly` spec to remove checkpoint injection logic

## Capabilities

### New Capabilities
- `chat-history`: Server-side conversation turn storage with save/load API and 8h TTL matching auth sessions

### Modified Capabilities
- `streaming-chat`: Remove checkpoint button and checkpoint lifecycle from chat header controls
- `context-assembly`: Remove checkpoint summary injection from `buildInjectedContext`; sticky notes remain
- `conversation-checkpoints`: **REMOVE** — replaced by server-side persistence

## Impact

- **Backend**: New `ChatStore` interface in `internal/auth/` or `internal/chat/`, new REST handlers, wired in `cmd/gateway/main.go`
- **Frontend**: `chat-assistant.tsx` saves turns on each exchange, loads on mount; checkpoint code removed
- **Helm**: No changes (in-memory, no new infrastructure)
- **Storage**: In-memory only — both auth sessions and chat history migrate to durable storage together in a future change
