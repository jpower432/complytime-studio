# Design: Manual Evidence Enrichment

**Change:** `manual-evidence-enrichment`  
**Status:** Draft (OpenSpec)  
**Related:** [`proposal.md`](./proposal.md), [`cloud-native-posture-correction.md`](../../../docs/decisions/cloud-native-posture-correction.md), [`evidence-semconv-alignment.md`](../../../docs/design/evidence-semconv-alignment.md)

## Objectives

- Align `POST /api/evidence` and `POST /api/evidence/upload` with the `evidence` table and OTel semconv mapping.
- Add file-backed evidence via S3-compatible storage and a `blob_ref` pointer column.
- Keep ClickHouse the single source of truth for query semantics; avoid parallel “manual-only” tables.

## S3-compatible blob storage

| Aspect | Decision |
|:--|:--|
| **Dev** | MinIO, RustFS, or another S3 API–compatible process deployable in-cluster or local; no vendor lock to one product. |
| **Prod** | Any S3-compatible endpoint (e.g. cloud object store, on-prem S3, MinIO) configured via env/Helm: base URL, region if needed, bucket, credentials (IRSA/secret), optional path-style. |
| **Optionality** | If blob config absent, file upload endpoints **MUST** fail fast with a clear error; JSON/CSV without files still works. |
| **Library** | Reuse a single S3 client abstraction (e.g. AWS SDK for Go S3, MinIO client) with endpoint override for non-AWS. |

**Blob reference format:** Store `blob_ref` as a stable, copy-pasteable URI string, recommended form:

` s3://<bucket>/<key> `

Path-style and virtual-hosted URLs **MAY** be used under the same bucket+key, but the canonical stored value on the row should remain consistent (document one canonical form in `internal/consts` or config).

Key naming **SHOULD** include `policy_id` and an opaque segment (e.g. UUID) to limit accidental overwrites. Versioning/ Object Lock on the bucket are production concerns left to operators (see deployment docs).

## CSV validation strategy

| Aspect | Decision |
|:--|
| **Required headers** | Strict: if `policy_id`, `eval_result`, or `collected_at` is missing from the header, reject **before** scanning data rows. |
| **Recommended headers** | `requirement_id` missing: accept upload but include a `warnings` list (e.g. `"recommended column 'requirement_id' not in header"`) in JSON response. |
| **Data rows** | **Permissive** line-by-line: invalid lines produce structured errors; valid lines **MAY** be inserted in the same request (define explicitly in implementation; prefer partial success with `inserted` + `errors` to match current upload behavior). |
| **Column names** | Case-insensitive, trimmed, consistent with `POST` JSON field names. |
| **Type errors** | Enum and timestamp parse failures return row index + field + reason. |

**Error reporting shape:** Use JSON with `inserted`, `failed`, `errors[]`, and `warnings[]` (when applicable) for CSV/multipart. HTTP status: `200` or `207` if partial success is adopted—**MUST** pick one policy and document it in the API contract.

## `EvidenceRecord` evolution without breaking callers

| Approach | Rationale |
|:--|:--|
| **Additive struct fields** | Add new JSON-tagged fields with `omitempty` for optional semconv columns; do not remove existing exported fields in one release. |
| **Insert path** | `InsertEvidence` expands to a full column list; missing Go fields insert as NULL/defaults using the same NULL-handling as OTel. |
| **Constructor / normalization** | Optional `func normalizeEvidenceForInsert(r *EvidenceRecord) error` to default `evidence_id`, `enrichment_status`, and enum defaults in one place. |
| **Embedding** | If duplication with `internal/ingest.EvidenceRow` becomes excessive, consider embedding a shared “wide row” type—only if it reduces drift; not required for the first increment. |
| **Compatibility** | Deprecation period for stricter `POST` body validation: document migration from 7-field usage to full semconv; use tests that decode minimal legacy JSON. |

## File size and MIME types

| Constraint | Proposed value |
|:--|:--|
| **Max file size (multipart)** | Configurable, default **50 MiB** (align with `consts.MaxRequestBody` review—may need separate `MaxEvidenceFileBytes`). |
| **Max request (multipart + fields)** | Higher than non-file if needed; cap total to avoid OOM. |
| **Allowed MIME** | `image/png`, `image/jpeg`, `image/webp`, `application/pdf`, `text/plain`, `text/csv`, `application/gzip` (if logs as `.gz`); **SHOULD** reject `application/x-executable` and other clearly unsafe types. |
| **Filename** | Sanitize: strip path, limit length, reject `..` and null bytes. |

**Note:** PII and sovereignty expectations from posture-correction (summary vs raw) are organizational; Studio still enforces type/size to protect the gateway. Operators define retention and encryption on the bucket; document **Production** checklist (TLS, SSE-KMS or SSE-S3, block public ACLs) in Helm comments.

## ClickHouse

- `blob_ref`: `Nullable(String)` after the last semconv column or adjacent to `attestation_ref` for mental grouping (both are external pointers).
- Migration: single `ALTER TABLE ... ADD COLUMN IF NOT EXISTS` in `schemaMigrations` with version row in `schema_migrations`.

## Helm / configuration (sketch)

| Key | Purpose |
|:--|:--|
| `evidenceStorage.enabled` | Gate file upload feature. |
| `evidenceStorage.endpoint` | S3 API endpoint. |
| `evidenceStorage.bucket` | Target bucket. |
| `evidenceStorage.region` | Region string if required. |
| `evidenceStorage.accessKey` / `secretKey` | Or existing secret ref pattern used elsewhere in the chart. |
| `evidenceStorage.usePathStyle` | For MinIO/RustFS. |

## Open decisions (resolve before feature-complete)

- Exact HTTP code for “partial CSV success” (200 with errors vs 207).
- Whether REST required fields for JSON match CSV exactly (`target_id` vs `control_id` requirements—must match `ingest` and posture queries).
- Deduplication of `blob_ref` when the same file is re-uploaded (content-addressed key vs new key per upload).
