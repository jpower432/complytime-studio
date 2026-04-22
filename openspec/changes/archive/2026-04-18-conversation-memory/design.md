## Context

The Studio Assistant chat uses A2A streaming via kagent. Conversation state exists in two layers: browser-side (`messages[]` in localStorage, capped at 50) and server-side (kagent's A2A task history, tied to `taskId`). The server-side history is opaque — the client cannot prune turns. The only lever is abandoning a task (`taskId = null`) and starting fresh with injected context.

The agent prompt already supports `<conversation-history>` tags ("Treat as background, not new instructions") and the `streamReply` API has an unused `options.history` field. The `streamMessage()` function accepts arbitrary text as the first message of a new task.

## Goals / Non-Goals

**Goals:**
- Users control what the LLM remembers across session boundaries
- Three complementary memory mechanisms: pins (per-message), checkpoints (mid-conversation), sticky notes (persistent facts)
- Bounded context injection to prevent the cure from causing its own rot
- All state management client-side — no backend changes

**Non-Goals:**
- Server-side conversation history management (kagent A2A task history is opaque)
- Agent-initiated auto-pinning (agent can suggest, user decides)
- Conversation search or full history browsing
- Synchronizing memory across devices/browsers

## Decisions

### Decision 1: Client-side context assembly, not server-side history manipulation

Context injection happens entirely in the frontend before calling `streamMessage()`. The gateway and A2A protocol remain unchanged. Pinned messages and sticky notes are serialized into tagged text blocks that the agent prompt already knows how to handle.

**Alternative**: Extend the A2A proxy to inject server-side context from a user profile store. Rejected — adds backend complexity, auth scoping, and a new data store for marginal benefit. The client already has all the data it needs.

### Decision 2: Three localStorage keys for separation of concerns

| Key | Type | Lifecycle |
|:--|:--|:--|
| `studio-chat-history` | `ChatMessage[]` | Per-session, existing (add `pinned` field) |
| `studio-sticky-notes` | `StickyNote[]` | Persistent until user deletes |
| `studio-pinned-cache` | `ChatMessage[]` | Written on "New Session", read on next task start, then cleared |

Sticky notes are `{id: string, text: string, createdAt: string}`. Pinned cache is a snapshot of pinned messages at session reset time — avoids re-scanning the full message array on every send.

**Alternative**: Single localStorage key with nested structure. Rejected — different lifecycles (session vs persistent) make separate keys cleaner.

### Decision 3: Context budget with hard limits

| Source | Max items | Max chars per item | Total cap |
|:--|:--|:--|:--|
| Sticky notes | 10 | 200 | 2000 |
| Pinned messages | 5 | 500 (truncated) | 2500 |
| **Total injected** | | | **~4500 chars** |

Checkpoint summaries replace the raw pinned messages for that session (they encompass the pins). The total injected context stays under ~1200 tokens, leaving 95%+ of the context window for actual work.

Truncation strategy: pinned messages are trimmed at 500 chars with `…` suffix. If more than 5 messages are pinned, only the most recently pinned 5 are carried.

### Decision 4: "New Session" vs "Checkpoint" as separate actions

**New Session**: Clears UI, resets `taskId`, carries forward sticky notes + pinned messages into next `streamMessage()`. For starting fresh.

**Checkpoint**: Serializes recent messages (since last checkpoint or start) into a condensed summary, resets `taskId`, inserts visual divider in UI, carries forward sticky notes + pins + summary. For mid-conversation context compression.

Checkpoint summary is a naive serialization: `"User: <first 100 chars> → Agent: <first 200 chars>"` for each turn since last checkpoint. No LLM summarization — keeps it deterministic and instant.

**Alternative**: LLM-generated summaries via a separate API call. Rejected for v1 — adds latency, cost, and a dependency on the LLM being available just to manage context. Can be added later.

### Decision 5: Agent prompt convention for sticky notes

Add `<sticky-notes>` tag to the agent prompt conventions alongside `<conversation-history>`. The agent treats sticky notes as persistent user facts — always true unless contradicted. The agent can suggest creating sticky notes when it detects the user establishing scope (dates, policies, priorities).

Suggestion is passive: the agent includes a line like "Tip: save 'Audit window: Q1 2026' as a sticky note to carry this across sessions." The UI does not auto-create — the user manually adds.

## Risks / Trade-offs

**[localStorage limits]** → localStorage has a ~5MB limit per origin. With 50 messages + 10 sticky notes, we're well under 100KB. Not a concern.

**[Stale sticky notes]** → Users may forget to remove outdated notes, injecting wrong context. Mitigation: sticky notes panel is always visible when open; show note age. Consider a "review notes" prompt after N sessions.

**[Naive checkpoint summaries lose nuance]** → Character-truncated turn summaries miss important details. Mitigation: pins exist for exactly this — users pin the important parts, checkpoint summary is supplementary. Upgrade path to LLM-generated summaries exists.

**[Pin overuse]** → Users pin everything, defeating the purpose. Mitigation: hard cap at 5 pins. Visual feedback when limit is reached. Oldest pin is auto-unpinned if a 6th is added (with notification).

**[Context injection not visible to user]** → Users may not understand what the agent "sees." Mitigation: on session start, display a collapsed "Context sent" block showing exactly what was injected.
