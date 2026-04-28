# Certifier workflow — design

## Context

The compliance manager reviews agent-produced audit logs (drafts), certifies them (promotes to history), and exports them for the auditor. This workflow currently spans Inbox (draft review), Posture drill-down (requirements/evidence context), and History (promoted records). The user mentally stitches these together.

## Decision 1: Audit workspace as a top-level view

**Choice:** Add `"audit"` to the `View` type union. Route: `#/audit/{id}` where `id` is a `draft_id` or `audit_id`. The workspace fetches the artifact by ID and renders the full review/export interface.

**Rationale:** A top-level route means the workspace is deep-linkable, bookmarkable, and can be opened from any entry point (Inbox, History, chat agent artifact callback). It follows the same pattern as `#/posture/{policy_id}`.

**Consequences:** New view type, new component, new route parsing. Sidebar does not need an "Audit" nav item — the workspace is reached contextually from Inbox and History, not from the sidebar.

## Decision 2: Two-panel layout

**Choice:** Left panel (60% width): audit result cards with type override dropdowns, reviewer notes, and agent reasoning. Right panel (40% width): tabbed policy context (Requirements, Evidence, History) scoped to the audit's policy and time window.

**Rationale:** The compliance manager needs to see both the agent's findings AND the underlying evidence simultaneously. Today this requires two browser tabs or constant navigation. Side-by-side eliminates the context switch.

**Consequences:** The right panel reuses `RequirementMatrixView`, `EvidenceView`, and `AuditHistoryView` with `policyIdOverride` props (already supported). No new data fetching logic needed.

## Decision 3: Lifecycle-aware mode

**Choice:** The workspace detects whether the artifact is a draft (`pending_review`) or promoted. In draft mode: edits enabled, "Save to History" button visible. In promoted mode: read-only, export buttons prominent.

**Rationale:** Same component, same URL pattern, different interaction mode. Avoids duplicating the UI for two lifecycle states.

**Consequences:** The workspace needs to try both `GET /api/draft-audit-logs/{id}` and `GET /api/audit-logs/{id}` (or a unified endpoint). If the draft is already promoted, it falls back to the audit log.

## Decision 4: Inbox becomes a launcher, not a workspace

**Choice:** Clicking a draft card in the Inbox navigates to `#/audit/{draft_id}` instead of expanding an inline detail panel. The inline `draft-detail` section is removed from `InboxView`.

**Rationale:** The Inbox is a triage surface — scan, prioritize, act. The act should be "open in workspace," not "review inline in a cramped panel."

**Consequences:** `InboxView` becomes simpler (list only, no detail panel). The `DraftReviewView` component is either refactored into the workspace or deprecated.

## Decision 5: History links to workspace in read-only mode

**Choice:** Clicking an audit card in `AuditHistoryView` navigates to `#/audit/{audit_id}` instead of expanding inline. The workspace renders in read-only mode with export actions.

**Rationale:** Consistent entry point. Whether the compliance manager or auditor reaches an audit from Inbox or History, they land in the same workspace.

**Consequences:** `AuditHistoryView` inline detail panel is removed. Audit detail is always the workspace.

## Related documents

- `openspec/changes/simple-rbac/` — reviewer role gating applies to workspace (read-only for reviewers)
- `openspec/changes/posture-card-clickable/` — consistent card-click navigation pattern
