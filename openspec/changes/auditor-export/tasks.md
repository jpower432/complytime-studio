# Tasks: `auditor-export`

## CSV export endpoint

- [x] Add `GET /api/export/csv` handler: validate `policy_id`, `audit_start`, `audit_end` (and optional `audit_id` if design adopts it).
- [x] Implement shared "requirement matrix row" query layer; ensure CSV column order and headers match the matrix view contract.
- [x] Emit generation metadata (preamble/comment convention per spec).
- [x] Set `Content-Type`, `Content-Disposition`, `Cache-Control: no-store`.
- [x] Register route in `internal/store` (or dedicated export package) and wire dependencies from `Store` + policy lookup for version.
- [x] Add structured logging: policy id, window, row count, duration (no PII in evidence payload).

## Excel export endpoint

- [x] Add `GET /api/export/excel` handler with same query validation as CSV.
- [x] Create workbook with sheets: **Executive Summary**, **Requirement Detail**, **Evidence Inventory**, **Gap List**; apply minimal header styling.
- [x] Pull aggregates for Executive Summary from same query surface as posture/matrix.
- [x] Optional: merge agent narrative from `audit_logs.summary` when a matching log exists.
- [x] Enforce size/timeout limits; return clear errors when exceeded.

## PDF export endpoint

- [x] Add `GET /api/export/pdf` handler with same scoping validation.
- [x] Render server-side PDF: cover/metadata page, summary, requirement table, gap section.
- [x] Document and implement truncation rules for very large row sets.
- [x] Optional: include agent narrative on first pages when available.

## Workbench export UI

- [x] Add export control to requirement matrix toolbar: format choice (CSV / Excel / PDF).
- [x] Pass current `policy_id` and selected audit window from view state.
- [x] Trigger download via `fetch` + blob; handle error states (4xx/5xx to toast).
- [x] Show spinner/disable during long exports.

## Agent narrative integration

- [x] Define selection rule: latest `AuditLog` for `policy_id` within window, or explicit `audit_id` param.
- [x] Map `audit_logs.summary` (and any future fields) into Excel Executive Summary and PDF intro; label as agent-generated.
- [x] Ensure exports succeed when summary is empty.

## Testing

- [x] Unit tests: query builders / row mapping for one policy and one window.
- [x] Integration tests (ClickHouse testcontainer or existing harness): empty window, all gaps, mixed evidence, "large" batch within test limits.
- [x] Parse generated CSV, `.xlsx` (excelize read-back), and PDF (page count or text extraction) in smoke assertions.
- [x] Test `Content-Disposition` filename sanitization.
- [x] Test auth: unauthenticated request when OAuth enabled returns `401/403` per existing middleware.
