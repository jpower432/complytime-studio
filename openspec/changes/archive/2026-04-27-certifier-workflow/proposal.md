## Why

The compliance manager's core loop — triage, review, certify, export — is split across three separate views with different mental models:

| Step | Current location | Mental model |
|------|-----------------|--------------|
| See posture changes | Inbox (notifications) | Feed of events |
| Review draft audit log | Inbox (inline detail panel) | Email-like read/act |
| See requirement detail | Posture → drill-down → Requirements tab | Hierarchical browse |
| Export audit | Requirement Matrix → Export toolbar | Buried inside drill-down |
| View audit history | Posture → drill-down → History tab | Separate tab, same drill-down |

The compliance manager must context-switch between feed (Inbox), hierarchy (Posture drill-down), and review (inline draft panel) to complete a single audit cycle. Draft review lives inside the Inbox but promoted audits live inside the History tab — same artifact, different locations depending on lifecycle state.

The QE review flagged this as a "should-fix": workable today but friction compounds with multiple policies.

## What Changes

- **Unified audit workspace** — a new top-level view (`#/audit/{draft_id}` or `#/audit/{audit_id}`) that persists across the draft → review → promoted lifecycle. The Inbox surfaces items into it; the workspace handles review, certification, and export in one screen.
- **Audit workspace layout** — left panel: result cards with type overrides and reviewer notes (existing draft review UI). Right panel: tabbed Requirements/Evidence/History for the associated policy (existing drill-down content). Top bar: audit metadata, save/promote actions, export buttons.
- **Inbox becomes a launcher** — clicking a draft in the Inbox navigates to the audit workspace instead of expanding an inline panel. Notifications still link to posture drill-down.
- **History links back** — clicking an audit in the History tab opens the same workspace in read-only mode (promoted, no edits).

## Capabilities

### New Capabilities
- `audit-workspace`: Unified view for draft review, certification, and export. Accessible from Inbox, History, and direct URL.

### Modified Capabilities
- `inbox-view`: Draft cards navigate to audit workspace instead of inline expand. Notification cards unchanged.
- `audit-history-view`: Audit cards navigate to audit workspace in read-only mode.
- `draft-review-view`: Absorbed into audit workspace. Component may be deprecated or refactored as a sub-component.

## Impact

- **Workbench**: New `AuditWorkspaceView` component. Modified Inbox and History navigation. New route `#/audit/{id}`.
- **API**: No backend changes. All data already available via existing endpoints.
- **UX**: Compliance manager stays in one screen for the entire audit lifecycle instead of bouncing between three views.

## Constitution Alignment

### II. Composability First

**Assessment**: PASS

The workspace composes existing sub-views (result cards, requirement matrix, evidence, history) into a unified layout. No new data dependencies. Existing views remain independently accessible.

### III. Observable Quality

**Assessment**: PASS

Audit state transitions (draft → promoted) happen in the same screen. The compliance manager sees the full context at every step.
