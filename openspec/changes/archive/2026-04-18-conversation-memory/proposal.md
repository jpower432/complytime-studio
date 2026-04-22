## Why

Long conversations with the Studio Assistant degrade in quality as stale context accumulates ("context rot"). The LLM loses track of decisions, repeats itself, or contradicts earlier analysis. Today the only option is "Clear" — a full reset that destroys all context. Users have no way to selectively control what the agent remembers.

## What Changes

- **Pinned messages**: Users can pin important agent responses (gaps found, mapping results). Pinned messages survive session resets and are injected as `<conversation-history>` into the next A2A task.
- **Checkpoints**: Mid-conversation action that condenses recent turns into a summary, starts a fresh A2A task, and carries forward pins + summary. Prevents rot without losing key context.
- **Sticky notes**: Persistent user-curated facts (audit window, priority gaps, scope) stored separately from conversation. Always injected as `<sticky-notes>` context on every new A2A task. Survives across all sessions.
- **Context assembly**: On new task creation, the frontend assembles injected context from sticky notes + pinned messages + checkpoint summary, bounded by a token budget (~4500 chars max).
- **Agent prompt update**: System prompt recognizes `<sticky-notes>` tag and can suggest pins/sticky notes when the user establishes persistent facts.
- **Replaces "Clear" button** with "New Session" (reset with carry-forward) and "Checkpoint" (condense mid-conversation). Sticky notes toggle via a dedicated panel button.

## Capabilities

### New Capabilities
- `pinned-messages`: Message pinning, pin persistence in localStorage, pin serialization for context injection on session reset
- `conversation-checkpoints`: Mid-conversation checkpoint action, turn condensation, visual separator, fresh A2A task creation with summary carry-forward
- `sticky-notes`: Persistent note store, notes panel UI, always-inject on new tasks, CRUD operations
- `context-assembly`: Unified context builder that composes sticky notes + pins + checkpoint summaries into bounded `<sticky-notes>` and `<conversation-history>` blocks for `streamMessage()`

### Modified Capabilities
- `streaming-chat`: Replace "Clear" with "New Session" / "Checkpoint" / sticky notes toggle. Add pin button per agent message. Add `pinned` field to `ChatMessage`.
- `agent-spec-skills`: Agent prompt updated to recognize `<sticky-notes>` tag and suggest pins/notes when user establishes persistent facts.

## Impact

- **Frontend**: `chat-assistant.tsx` — new UI controls, message model change, context assembly logic. New sticky notes panel component. CSS additions.
- **A2A client**: `a2a.ts` — `streamMessage()` already accepts text; context assembly happens before the call. No API changes needed.
- **Agent prompt**: `agents/assistant/prompt.md` — add `<sticky-notes>` convention and suggestion behavior.
- **Storage**: localStorage only — no backend changes. Two new keys: `studio-sticky-notes`, existing `studio-chat-history` gets `pinned` field on messages.
- **No backend changes**: All memory management is client-side. The A2A protocol and gateway proxy are unchanged.
