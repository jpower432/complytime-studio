# ADR-001: In-Memory Session Persistence for Conversation History

**Status:** Accepted
**Date:** 2026-04-18

## Context

The chat assistant is stateless — conversation history lives in the browser's `localStorage`. Page refreshes, tab closures, and device switches wipe agent context. Client-side workarounds (checkpoints, sticky notes) partially mitigate this but require manual user action.

Users report the agent "forgets what I typed earlier" during normal workflows.

## Decision

Store conversation turns server-side in the gateway's in-memory session store, keyed by user email. Apply the same lifecycle as auth sessions.

| Dimension | Choice | Rationale |
|:--|:--|:--|
| Scope | Session persistence (not full archive) | Solves the immediate pain without building a conversation archive UI |
| Retention | 8h TTL, explicit clear on "New Session" | Matches `sessionMaxAge` for auth cookies |
| Visibility | Private per-user | Conversations are working scratchpad, not compliance artifacts |
| Storage | In-memory (`MemorySessionStore`) | Consistent with auth session storage; both move to durable storage together |

## Consequences

**Accepted tradeoffs:**
- Gateway pod restart clears conversation history (same as auth sessions today)
- Single-replica affinity — session lives in one gateway process
- No cross-device sync until durable storage is added

**Future:** When auth sessions move to durable storage (ClickHouse or Redis), conversation sessions move with them in the same change. This ADR does not prescribe the durable backend — only that both migrate together.

## Alternatives Considered

| Alternative | Rejected Because |
|:--|:--|
| Full conversation archive (ChatGPT-like) | Requires pagination, search, conversation list UI — excessive for current needs |
| ClickHouse-backed storage now | Adds a hard dependency for chat; inconsistent with auth session pattern |
| TTL-only (no explicit clear) | Users need a deliberate "start fresh" action |
| Shared/team-visible conversations | Privacy risk for drafts and sensitive reasoning; AuditLog artifacts already capture outputs that matter |
