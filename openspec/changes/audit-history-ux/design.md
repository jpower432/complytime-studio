# Design: Audit History UX

## Audit History View

### Card States

| State | Content | Interaction |
|:--|:--|:--|
| Collapsed | Date range, framework badge, strengths/findings/gaps counts, creator, date, expand indicator | Click header to expand |
| Expanded | Collapsed content + full YAML content (lazy-loaded) + "Open Workspace" button | Click header to collapse |

### Data Flow

- Collapsed: uses list endpoint data (`/api/audit-logs?policy_id=...`) — no `content` field
- Expanded: fetches full audit via `/api/audit-logs/{id}` on first expand, caches in component state
- Expand/collapse is local state (`expandedId` + `expandedContent`)

### Component Changes

`audit-history-view.tsx`:
- Add `expandedId` and `expandedContent` state
- `toggleExpand(log)`: if content available, expand immediately; otherwise fetch by ID
- Remove `navigateToAudit` as primary card action; move to explicit button inside expanded body
- Remove `cardKeyHandler` on card (no longer navigates); header is the click target

## Audit Workspace Right Panel

### Before

Three tabs embedding full views:
- RequirementMatrixView
- EvidenceView
- AuditHistoryView

### After

Compact `<aside>` with:
1. **Summary stats**: 3-column grid of strengths/findings/gaps counts
2. **Metadata list**: period, framework, creator, model, prompt version
3. **Navigation links**: buttons to Policy Detail tabs (Requirements, Evidence, History)

Links use `navigateToPolicy(policyId, tab)` to return to Policy Detail with correct tab active.

## CSS

- `.audit-list` replaces `.audit-card-list`
- `.audit-card-header` is the click target with hover highlight
- `.audit-card-body` has border-top separator, max-height 400px with overflow scroll
- `.workspace-panels` grid changes from `3fr 2fr` to `1fr 280px`
- `.workspace-right` becomes a fixed-width sidebar with metadata and links
