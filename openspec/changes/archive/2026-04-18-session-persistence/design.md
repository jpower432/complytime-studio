## Context

Chat history lives in `localStorage`. The agent is stateless via A2A — each `streamMessage` starts a new task, each `streamReply` continues one. Page refresh loses the `taskIdRef` and all displayed messages. Checkpoints and sticky notes are manual workarounds that don't solve the core problem.

Auth sessions already use an in-memory `MemorySessionStore` keyed by session ID with passive TTL expiration. The same pattern applies to conversation turns.

## Goals / Non-Goals

**Goals:**
- Conversation turns survive page refresh, tab close, and browser restart (within TTL)
- Agent receives prior turns as context on reconnect
- Remove checkpoint feature (superseded by server-side continuity)
- Keep sticky notes (orthogonal — user-curated persistent context)

**Non-Goals:**
- Full conversation archive with search/browse UI
- Multi-device sync (requires durable storage)
- Shared/team-visible conversations
- Durable storage backend (deferred — migrates with auth sessions later)

## Decisions

### 1. Store: Reuse MemorySessionStore pattern

Add a `ChatStore` interface alongside `SessionStore` in `internal/auth/`. Use the same `sync.RWMutex` + map pattern with TTL. Keyed by user email (one active session per user).

**Alternative considered:** Separate `internal/chat/` package. Rejected — the chat store shares lifecycle, TTL, and future migration path with auth sessions. Co-locating keeps the "migrate both together" constraint visible.

### 2. API: Two endpoints on the gateway

| Endpoint | Method | Body | Behavior |
|:--|:--|:--|:--|
| `/api/chat/history` | `GET` | — | Returns `{messages: [...], taskId: string\|null}` for the authenticated user |
| `/api/chat/history` | `PUT` | `{messages: [...], taskId: string\|null}` | Overwrites the user's conversation state |

PUT (full replace) over PATCH (incremental) because the client already holds the full message array. Simpler, no merge conflicts.

**Alternative considered:** `POST` per-turn append. Rejected — adds complexity for conflict resolution when the client already has the authoritative state.

### 3. Client: Save on every exchange, load on mount

- On component mount: `GET /api/chat/history` → hydrate `messages` and `taskIdRef`
- After each agent response completes (`onDone`): `PUT /api/chat/history` with current state
- On "New Session": `PUT /api/chat/history` with empty messages and null taskId

No debouncing needed — saves happen at exchange boundaries, not on every keystroke.

### 4. Context injection: Replace checkpoint with server history

Currently `buildInjectedContext` accepts `checkpointSummary`. After this change:
- Remove `checkpointSummary` parameter
- Server-side `taskId` means `streamReply` resumes the A2A task directly — the agent already has context from prior turns in the same task
- If `taskId` is null (new task after server restart), the first `streamMessage` injects sticky notes only — no synthetic history replay
- This is the accepted tradeoff: pod restart loses agent context, same as auth session loss

### 5. Remove checkpoints entirely

Checkpoints exist because the client had no persistence. With server-side storage:
- `taskId` persists across refresh → agent keeps its context natively
- Manual "condense and reset" is unnecessary
- Remove: checkpoint button, `handleCheckpoint`, checkpoint divider rendering, `CHECKPOINT_KEY`, `loadAndClearCheckpointSummary`, `saveCheckpointSummary`

## Risks / Trade-offs

| Risk | Mitigation |
|:--|:--|
| Pod restart clears chat history | Accepted — same as auth sessions. Both migrate to durable storage together (ADR-001) |
| Single-replica affinity | Accepted — gateway runs as single replica today. Multi-replica requires sticky sessions or shared store |
| PUT overwrites on stale client | Low risk — single user, single active tab. If two tabs race, last write wins (acceptable for scratchpad data) |
| Removing checkpoints loses "condense" ability | Sticky notes cover the "remember this" use case. Agent's own task context handles continuity |
