# Tasks — certifier workflow

## Routing

- [x] Add `"audit"` to the `View` type union in `app.tsx`.
- [x] Add route parsing for `#/audit/{id}` in `parseHash`. Extract `id` param.
- [x] Add `selectedAuditId` signal to `app.tsx`.
- [x] Add `navigateToAudit(id: string)` helper function.

## Audit workspace component

- [x] Create `audit-workspace-view.tsx` with two-panel layout: left panel (results) + right panel (policy context tabs).
- [x] Left panel: fetch draft/audit by ID. Render result cards with type override dropdowns, reviewer notes, agent reasoning.
- [x] Right panel: render `RequirementMatrixView`, `EvidenceView`, `AuditHistoryView` with `policyIdOverride` set to the audit's `policy_id`.
- [x] Top bar: audit metadata (policy, period, framework, model, created_at). Action buttons: Save to History (draft mode), Download YAML.
- [x] Lifecycle detection: try `GET /api/draft-audit-logs/{id}` first. If 404, try `GET /api/audit-logs/{id}`. Set mode (editable vs read-only) based on response.
- [x] Auto-save edits with debounce.
- [x] "Save to History" button triggers promote flow, then switches workspace to read-only mode without navigating away.

## CSS

- [x] Add `.audit-workspace` layout styles: two-panel grid, responsive collapse to stacked on narrow viewports.
- [x] Left panel scrolls independently from right panel.

## Inbox simplification

- [x] In `inbox-view.tsx`: change draft card `onClick` to `navigateToAudit(draft.draft_id)`.
- [x] Remove the inline `draft-detail` section from `InboxView`.
- [x] Remove `selected`, `results`, `edits`, `saveState`, `promoting` state and related logic from `InboxView`. Keep notification handling unchanged.

## History linking

- [x] In `audit-history-view.tsx`: change audit card `onClick` to `navigateToAudit(log.audit_id)`.
- [x] Remove inline audit detail expansion from `AuditHistoryView`.

## DraftReviewView deprecation

- [ ] Evaluate whether `draft-review-view.tsx` is still needed. If the audit workspace fully replaces it, remove the component and any remaining imports.

## App wiring

- [x] Add `{view === "audit" && <AuditWorkspaceView />}` to the main view switch in `App()`.
- [x] Breadcrumb navigation: workspace shows `Posture > {Policy} > Audit` with clickable links back.

## Tests

- [ ] Verify clicking inbox draft card navigates to `#/audit/{draft_id}`.
- [ ] Verify clicking history audit card navigates to `#/audit/{audit_id}`.
- [ ] Verify workspace loads in editable mode for `pending_review` drafts.
- [ ] Verify workspace loads in read-only mode for promoted audit logs.
- [ ] Verify "Save to History" transitions workspace from editable to read-only without page reload.
- [ ] Verify right panel tabs show policy-scoped requirements/evidence/history.
