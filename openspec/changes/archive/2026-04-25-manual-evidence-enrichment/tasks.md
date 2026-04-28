# Tasks: `manual-evidence-enrichment`

Execution groups for the change. Order within a group is flexible unless noted.

## Evidence struct enrichment

- [x] Expand `EvidenceRecord` in `internal/store/store.go` with semconv-aligned fields (`requirement_id`, `plan_id`, `engine_name`, `engine_version`, `confidence`, `compliance_status`, `enrichment_status`, `target_name`, `target_type`, `target_env`, `frameworks`, `remediation_*`, `attestation_ref`, `blob_ref`, etc.) matching ClickHouse and [`evidence-semconv-alignment.md`](../../../docs/design/evidence-semconv-alignment.md).
- [x] Update `ingestEvidenceHandler` validation to align with new required/optional rules (coordinate with CSV rules).
- [x] Extend `QueryEvidence` / `EvidenceFilter` as needed so Workbench and APIs return enriched fields; keep JSON stable with `omitempty` on new fields.
- [x] Add unit tests: JSON decode of minimal legacy payloads + full semconv payload.

## Blob storage integration

- [x] Introduce S3-compatible client abstraction (config from env, shared dial/retry).
- [x] Implement upload path: stream/multipart file -> object put -> return `s3://bucket/key` (or agreed canonical form).
- [x] Wire gateway to fail clearly when storage disabled but file part present.
- [x] Add Helm/values and Secret wiring for dev (MinIO/RustFS) and prod-shaped overrides.
- [x] Integration test with MinIO (or testcontainers) for happy path and failed put.

## CSV validation

- [x] Refactor `parseCSVEvidence` to require headers: `policy_id`, `eval_result`, `collected_at`.
- [x] Emit **warning** when `requirement_id` column absent from header (response field + logging).
- [x] Map all supported CSV columns to `EvidenceRecord` fields (same names as JSON).
- [x] Table-driven tests: missing required header, bad timestamp row, good partial rows, warn-only recommended.
- [x] Document CSV format in a short `docs/` or OpenSpec snippet (if project convention allows).

## Workbench upload form

- [x] Add UI fields: `requirement_id`, `plan_id`, and optional engine/confidence/compliance.
- [x] Add file attachment control; show max size and allowed types.
- [x] Surface server validation/warning messages (errors + warnings lists).
- [x] E2E or manual test checklist: submit enriched row with and without file.

## Schema migration

- [x] Add `blob_ref Nullable(String)` to `evidence` DDL in `EnsureSchema` for new clusters.
- [x] Add versioned migration in `schemaMigrations()` for existing clusters (`ALTER` add column idempotent).
- [x] Verify OTel/exporter path still inserts (NULL `blob_ref`) -- ingest writer updated.

## Testing

- [x] **Unit:** enum validation, `blob_ref` format, CSV header logic.
- [x] **Integration:** insert via REST vs fixture OTel row -> same `SELECT` columns; posture-style query on `requirement_id`.
- [x] **Contract:** golden JSON for `POST /api/evidence` and upload responses.

**Manual / E2E checklist:** [`workbench-upload-checklist.md`](./workbench-upload-checklist.md)

**CSV reference:** [`docs/design/evidence-csv-import.md`](../../../docs/design/evidence-csv-import.md)
