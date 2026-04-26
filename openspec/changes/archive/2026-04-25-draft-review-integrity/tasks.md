## 1. Schema & Store

- [x] 1.1 Add `reviewer_edits String DEFAULT '{}'` column to `draft_audit_logs` CREATE TABLE in `internal/clickhouse/client.go`
- [x] 1.2 Add `ReviewerEdits string` field to `DraftAuditLog` struct in `internal/store/store.go`
- [x] 1.3 Update `InsertDraftAuditLog` to include `reviewer_edits` column
- [x] 1.4 Update `ListDraftAuditLogs` and `GetDraftAuditLog` queries to SELECT `reviewer_edits`
- [x] 1.5 Add `UpdateDraftEdits(ctx, draftID, reviewerEdits string) error` method to `DraftAuditLogStore` interface and ClickHouse implementation

## 2. PATCH Handler

- [x] 2.1 Add `PATCH /api/draft-audit-logs/{id}` route in `registerStoreRoutes`
- [x] 2.2 Implement `updateDraftEditsHandler`: validate draft exists, status is `pending_review`, truncate notes to 2000 chars, call `UpdateDraftEdits`
- [x] 2.3 Return 409 if draft is already promoted, 404 if not found

## 3. Promote Merge

- [x] 3.1 Create `mergeReviewerEdits(content string, editsJSON string) (string, error)` in `internal/store/` that parses edits, walks YAML results, applies type overrides, appends reviewer-note fields
- [x] 3.2 Call `mergeReviewerEdits` in `PromoteDraftAuditLog` before inserting into `audit_logs`

## 4. Frontend Auto-Save

- [x] 4.1 Lift `overrideType` and `note` state from `ResultCard` into `DraftReviewView` as a `Record<string, { type_override: string, note: string }>` edits map
- [x] 4.2 On draft detail load, pre-fill edits map from `reviewer_edits` in the GET response
- [x] 4.3 Add debounced PATCH call (1s) triggered by edits map changes
- [x] 4.4 Add save indicator ("Saving..." / "Saved") in the detail header
- [x] 4.5 Pass edits + onChange callbacks down to `ResultCard` as props

## 5. Verification

- [x] 5.1 Override a result type, navigate away, reopen — verify override persists
- [x] 5.2 Add a note, refresh page, reopen — verify note persists
- [x] 5.3 Promote draft with overrides — verify official audit log content reflects overrides
- [x] 5.4 Attempt PATCH on promoted draft — verify 409 response
- [x] 5.5 TypeScript compiles clean
