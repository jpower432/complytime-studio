## Context

The Draft Review view lets reviewers reclassify agent results (e.g., Finding -> Strength) and add notes. These edits live in React `useState` — lost on navigation, refresh, or promote. The promote endpoint copies unmodified agent content to the official audit log table, discarding reviewer intent.

## Goals / Non-Goals

**Goals:**
- Persist reviewer edits (type overrides + notes) server-side so they survive navigation/refresh
- Merge edits into YAML at promote time so the official record reflects the reviewer's decisions
- Auto-save edits with debounce to eliminate explicit save friction

**Non-Goals:**
- Full collaborative editing (multi-user simultaneous review)
- Versioning / undo history of edits
- Changing the draft queue ingestion path (artifact interceptor is unchanged)

## Decisions

**1. Store edits as a JSON column, not as modified YAML**

Edits are stored in a `reviewer_edits String DEFAULT '{}'` column on `draft_audit_logs`. The JSON maps result IDs to `{ type_override, note }`. This avoids re-serializing YAML on every keystroke and keeps the original agent content intact for auditability.

Alternative: Modify the `content` column directly. Rejected — loses the original agent output, making it impossible to diff reviewer changes.

**2. PATCH endpoint for partial updates**

`PATCH /api/draft-audit-logs/{id}` accepts `{ reviewer_edits: {...} }`. This is idempotent and debounce-friendly. The handler validates that the draft exists and is still `pending_review`.

Alternative: PUT replacing the entire draft. Rejected — heavier payload, race-condition prone with auto-save.

**3. Merge at promote time**

`PromoteDraftAuditLog` reads `reviewer_edits`, walks the parsed YAML results, applies type overrides, appends notes, then serializes the merged content into the official `audit_logs` row. The original `content` column on the draft is never mutated.

**4. ClickHouse schema: ALTER TABLE ADD COLUMN**

ClickHouse MergeTree supports `ALTER TABLE ADD COLUMN` with a default. Existing rows get `'{}'`. No migration needed beyond the DDL in the schema init.

## Risks / Trade-offs

- **[Stale edit on concurrent promote]** If user A edits while user B promotes, edits are lost. Mitigation: promote checks `status = 'pending_review'`; once promoted, PATCH returns 409 Conflict.
- **[Large reviewer_edits JSON]** Unbounded notes could grow large. Mitigation: truncate notes to 2000 chars in the PATCH handler.
- **[ClickHouse String column for JSON]** No native JSON type in ClickHouse Enum-era schema. Mitigation: parse/validate in Go handler, keep the column opaque to queries (no JSON path filters needed).
