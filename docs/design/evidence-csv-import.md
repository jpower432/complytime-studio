# Evidence CSV Import

> **Status:** Removed. CSV/multipart upload was removed during the PostgreSQL migration. Evidence ingestion is now exclusively via `POST /api/ingest` using Gemara EvaluationLog/EnforcementLog YAML format.

See `internal/store/ingest_handler.go` for the current ingestion path.
