## Why

Reviewer edits (type overrides, notes) in the Draft Review view are local `useState` — they vanish on navigation or refresh. The "Save to History" button promotes the *original* agent content, discarding all reviewer work. This breaks the human-in-the-loop audit workflow: reviewers reclassify findings and add justifications, but none of it persists.

## What Changes

- Add `PATCH /api/draft-audit-logs/{id}` endpoint to persist reviewer edits (type overrides, notes) back to the draft row before promotion
- Track per-result reviewer overrides and notes in a `reviewer_edits` JSON column on `draft_audit_logs`
- Merge reviewer edits into the YAML content at promote time so the official audit log reflects the reviewer's final decisions
- Add auto-save (debounced) in the Draft Review UI so edits persist without an explicit save button

## Capabilities

### New Capabilities
- `draft-reviewer-edits`: Backend storage and API for persisting reviewer type overrides and notes on draft audit log results

### Modified Capabilities
- `react-workbench`: Draft Review UI auto-saves reviewer edits and displays persisted state on reload
- `streaming-chat`: No changes needed (chat save routes to drafts via existing artifact interceptor)

## Impact

- `internal/store/handlers.go` — new PATCH handler
- `internal/store/store.go` — `DraftAuditLogStore` interface gains `UpdateDraftEdits` method
- `internal/clickhouse/client.go` — schema adds `reviewer_edits` column, `PromoteDraftAuditLog` merges edits into content
- `workbench/src/components/draft-review-view.tsx` — auto-save, load persisted edits, display saved state
