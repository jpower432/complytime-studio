## Why

The REST and CSV evidence upload paths populate a subset of columns compared to the OTel path. `POST /api/evidence` accepts 7 fields (`evidence_id`, `policy_id`, `target_id`, `control_id`, `rule_id`, `eval_result`, `collected_at`). The full `evidence` table has 30+ columns including `requirement_id`, `plan_id`, `engine_name`, `confidence`, `compliance_status`, and `attestation_ref`.

Teams that don't run OTel collectors — the majority of analysts migrating from Hyperproof — upload spreadsheets and attach files (screenshots, PDFs, logs). Their evidence arrives without requirement linkage, plan association, or enrichment metadata. This makes manually-ingested evidence invisible to posture checks (which match on `requirement_id`) and produces incomplete AuditLogs.

The [cloud-native posture correction](../../docs/decisions/cloud-native-posture-correction.md) elevates manual ingest to first-class status.

## What Changes

- **Enriched REST handler**: `POST /api/evidence` and `POST /api/evidence/upload` accept all semconv-aligned columns. At minimum: `requirement_id`, `plan_id`, `engine_name`, `confidence`, `compliance_status`, `target_name`, `target_type`, `target_env`.
- **File evidence storage**: New S3-compatible blob storage for screenshots, PDFs, and log files. Evidence rows reference file artifacts via a `blob_ref` column (bucket + key).
- **CSV column mapping**: Upload handler validates and maps CSV columns to evidence table columns. Rejects uploads with missing required fields (`policy_id`, `eval_result`, `collected_at`). Warns on missing recommended fields (`requirement_id`).
- **Ingestion parity**: Both REST/CSV and OTel paths produce identical evidence rows. Downstream processing (posture checks, audit workflows) operates identically regardless of ingestion path.

## Capabilities

### New Capabilities
- `file-evidence-storage`: S3-compatible blob storage for file-based evidence artifacts with `blob_ref` pointer in ClickHouse
- `csv-column-validation`: Upload handler validates CSV structure against evidence schema, rejects on missing required columns, warns on missing recommended columns

### Modified Capabilities
- `evidence-ingestion`: REST and CSV handlers accept all semconv-aligned columns instead of the current 7-field subset
- `evidence-semconv-alignment`: Add `blob_ref Nullable(String)` column to the evidence table

## Impact

- **Gateway**: `internal/store/handlers.go` — enriched `EvidenceRecord` struct and `InsertEvidence` query. New file upload handler writing to S3-compatible storage.
- **ClickHouse**: New `blob_ref` column on `evidence` table (nullable, additive migration).
- **Helm**: Optional S3-compatible storage configuration in `values.yaml` (endpoint, bucket, credentials).
- **Agent**: No changes — existing queries gain richer data from manually-ingested evidence.
- **Workbench**: Evidence upload form gains fields for `requirement_id`, `plan_id`, and file attachment.

## Constitution Alignment

### I. Autonomous Collaboration

**Assessment**: PASS

Enriched evidence rows are self-describing. File evidence is content-addressed in blob storage. No coordination required between upload and downstream processing.

### II. Composability First

**Assessment**: PASS

Blob storage is optional — evidence without files works identically. CSV validation is a gateway concern, not a schema concern.

### III. Observable Quality

**Assessment**: PASS

Every upload produces a complete evidence row with provenance. Missing fields are explicit NULLs, not silent omissions. CSV validation errors return structured feedback.

### IV. Testability

**Assessment**: PASS

Upload handlers testable with fixture CSVs. Blob storage testable with MinIO in integration tests. Enrichment parity verifiable by comparing OTel and REST rows for the same evidence.
