# Evidence CSV import

Manual bulk import uses `POST /api/evidence/upload` with a `file` form field (`.csv` or `.json`).

## Required columns

| Column | Format |
|:-------|:-------|
| `policy_id` | string |
| `eval_result` | ClickHouse enum: `Not Run`, `Passed`, `Failed`, `Needs Review`, `Not Applicable`, `Unknown` |
| `collected_at` | RFC3339 timestamp |

## Recommended columns

| Column | Response |
|:-------|:---------|
| `requirement_id` | If the header omits this column, the API returns **warnings** (response `warnings` + log); rows still import when other fields validate. |

## Optional columns

Row values map to `EvidenceRecord` / ClickHouse `evidence` columns with the same names, including: `target_id`, `target_name`, `target_type`, `target_env`, `engine_name`, `engine_version`, `rule_id`, `rule_name`, `control_id`, `control_catalog_id`, `control_category`, `plan_id`, `confidence`, `compliance_status`, `risk_level`, `enrichment_status`, `eval_message`, `evidence_id`, `attestation_ref`, `source_registry`, `blob_ref`.

## Row validation

- Invalid `collected_at` on a line adds an entry to response `errors` and skips that line.
- Rows missing `policy_id`, `target_id`, `control_id`, or `collected_at` after parse are counted in `failed` and not inserted.

See `parseCSVEvidence` in `internal/store/handlers.go`.
