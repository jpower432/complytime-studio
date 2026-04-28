# Workbench manual evidence — manual / E2E checklist

**Evidence view → Upload Evidence → Manual Entry**

| Step | Action | Expected |
|:-----|:-------|:---------|
| 1 | Fill policy, target ID, control ID, rule ID; leave attachment empty; Submit | `Inserted 1 row(s)`; row appears in table after search |
| 2 | Same with optional requirement ID / plan ID | Same |
| 3 | Attach allowed file (e.g. PDF); Submit with blob storage **disabled** | HTTP 400; message includes `file upload not supported: blob storage not configured` |
| 4 | Attach file with blob storage **enabled** (BLOB_* env) | Success; inserted row has `blob_ref` `s3://...` when queried via API |
| 5 | CSV upload path | Warnings if `requirement_id` column missing; `inserted` / `failed` match expectations |

**Regression:** compliance status dropdown values match ClickHouse enums (`Compliant`, `Non-Compliant`, `Exempt`, `Not Applicable`, `Unknown`).
