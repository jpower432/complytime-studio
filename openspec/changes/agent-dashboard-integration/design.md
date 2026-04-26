# Design: Agent–Dashboard Integration

## Context

ComplyTime Studio repositions the workbench: structured views (posture, requirement matrix, evidence) are the primary analyst surface; the chat overlay augments with synthesis and deep questions. The assistant continues to use A2A, MCP, and the existing HITL model without prompt or tool rewrites. This document records implementation decisions for PostureView data plumbing, context injection, canned queries, real-time view updates, and migration from the prior agent-summary posture UI.

## Decision 1: PostureView data source

**Choice:** PostureView reads posture exclusively via a **gateway REST endpoint** that queries the ClickHouse `policy_posture` view server-side. The Preact workbench does **not** connect to ClickHouse or MCP directly.

**Alternatives considered:**
- **Direct ClickHouse from the browser** — Rejected. Exposes no viable secure path: credentials, SQL surface, and network boundaries belong on the gateway.
- **Agent-mediated fetch for the grid** — Rejected. Conflicts with dashboard-first goals; adds latency and couples UI refresh to model availability.
- **Optional second REST service** — Rejected. Duplicates the gateway’s role as the user-facing API and `internal/store` boundary.

**Rationale:** Architecture already funnels data access through the Go gateway and its ClickHouse stores. A single `GET /api/posture` handler reuses the same authz and transport as other `/api` routes, keeps the `policy_posture` contract in one place, and allows caching or shaping later without workbench changes.

**Migration note:** The proposal also names `unified_compliance_state` for drill-down; the first REST slice may scope to `policy_posture` with follow-on query parameters or companion endpoints as the requirement matrix work lands, without changing the "no direct ClickHouse in browser" rule.

## Decision 2: Context injection mechanism

**Choice:** **Extend the existing `buildInjectedContext` + `buildDashboardContext` pattern** in `workbench/src/components/chat-assistant.tsx`. Add keys for the active view (e.g. `control_id`, `requirement_id`, evidence filter object) as optional string fields in the `Record<string, string>` or a small typed structure serialized to JSON in the same bracket as today’s `Dashboard context: {...}`.

**Alternatives considered:**
- **New parallel injection pipeline** (e.g. separate header-only channel) — Rejected. Splits what the agent sees and complicates the A2A client.
- **Base64 or opaque blob only** — Rejected. Harder to debug and to extend per view; JSON keeps parity with the prompt’s instruction to use structured context.

**Rationale:** `buildInjectedContext` already prepends non-interactive context to the first `streamMessage` on a new task. Extending the record preserves behavior for resumed tasks (which use `streamReply` without re-injecting) and matches architecture.md’s continuity story. Per-view components set shared signals (e.g. existing `selectedPolicyId` + new atoms for control/evidence filters) that `buildDashboardContext` reads when composing the next send.

**Consequence:** Views that have not yet wired new keys simply omit them; the agent prompt’s "ask once" behavior remains when policy or window is still missing.

## Decision 3: Canned query buttons — static vs configurable

**Choice:** **Phase 1: static list in the workbench** (constants next to `ChatAssistant` or a small `canned-queries.ts` module). Copy maps one-to-one to the assistant’s routing (posture check, audit production, gap summary). **Phase 2 (optional):** read labels or enabled flags from a gateway `GET /api/config` (or an extension of the existing config payload) if operations need to disable a workflow or adjust wording without a frontend build.

**Alternatives considered:**
- **All-config driven from day one** — Deferred. Empty benefit until multiple deployments need different canned sets; adds contract surface before usage.
- **Server-pushed A2A "quick actions"** — Rejected for now. Heavier protocol change; static buttons satisfy the proposal’s learning-curve goal.

**Rationale:** Canned actions are pre-formed user messages plus the same injection path as manual sends. Static strings align with the prompt’s fixed workflow names and keep CI deterministic.

## Decision 4: Real-time updates — SSE and dashboard subscription

**Facts:** The gateway already proxies A2A with an SSE response and can auto-persist `AuditLog` and similar artifacts to ClickHouse (see architecture.md: artifact persistence interceptor).

**Choice:** **Workbench subscribes to success paths it already has:** (1) **artifact callbacks** in `StreamCallbacks.onArtifact` — after a relevant artifact, trigger a **narrow refetch** of the list endpoint used by the active view (e.g. `GET /api/audit-logs` for history) or emit a small app-level event (e.g. `postureInvalidate`) that PostureView listens for. (2) For **server-side** persistence the user did not stream locally (e.g. another tab), full cross-tab real-time is **out of scope** unless the gateway adds a generic SSE fan-out; v1 may rely on **focus refetch** or **polling** on a long interval for Audit History.

**Alternatives considered:**
- **New WebSocket for all table changes** — Rejected for this change; scope creep.
- **Only manual refresh** — Rejected. Conflicts with "real time via existing SSE" in the proposal for the same session that produced artifacts.

**Rationale:** Same-session auto-persist already implies the client saw the stream; hooking `onArtifact` + persisted entity type is the minimal bridge. Posture aggregates may refresh on artifact types that impact `policy_posture` materialization (e.g. after `EvidenceAssessment` processing) using the same event or a debounced `GET /api/posture` refetch.

## Decision 5: Migration path from agent-summary PostureView

**Steps (conceptual order):**
1. **Implement** `GET /api/posture` and contract tests with seeded `policy_posture` data.
2. **Add** a PostureView branch (or feature flag) that renders from REST JSON only; keep the old card component unused behind the flag for one release if rollback is needed.
3. **Remove** agent-summary cards and any code paths that parse chat or AuditLog text for the main posture grid.
4. **Verify** with manual and automated tests: Posture with agent offline; chat overlay still works for "why" questions.
5. **Document** the behavioral change in the change proposal and release notes (dashboard-first).

**Roll-forward criteria:** E2E or integration test: load PostureView → `GET /api/posture` called → no dependency on a prior `streamMessage` for numeric posture display.

**Rollback:** Re-enable the flag to show the legacy UI (only if the flag was retained); production preference is to fix forward on REST failures rather than reintroduce agent cards as the source of truth.

## Risks and mitigations

| Risk | Mitigation |
|:---|:---|
| `policy_posture` empty or slow on large tenants | Server-side `LIMIT`/`WHERE` and indexes per evidence-attestation-pipeline design; consider MV promotion later. |
| Context JSON grows large | Cap verbose filters in `buildDashboardContext` or truncate with ellipsis in the injected string while keeping IDs. |
| Double-fetch on every artifact | Debounce refetch; only refetch views currently mounted. |

## Non-goals (this change)

- Changing agent prompts, skills, or MCP allowlists.
- New ClickHouse schema for posture (rely on existing `policy_posture` / migrations).
- Cross-tab real-time without new gateway channels (explicitly optional follow-up).
