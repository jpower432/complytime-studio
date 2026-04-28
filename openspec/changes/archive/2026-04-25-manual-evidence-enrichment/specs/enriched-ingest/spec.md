# Delta Spec: Enriched Ingest (Manual Evidence)

**Change:** `manual-evidence-enrichment`  
**Capability:** `enriched-ingest` (delta over `evidence-ingestion`)

Terms use [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119) semantics.

---

## REQ-1: `POST /api/evidence` accepts semconv-aligned body fields

The gateway **MUST** accept a JSON array where each object **MAY** include all attributes that map to the `evidence` table per [`docs/design/evidence-semconv-alignment.md`](../../../../../docs/design/evidence-semconv-alignment.md), including at minimum the following request keys where applicable: `evidence_id`, `policy_id`, `target_id`, `target_name`, `target_type`, `target_env`, `engine_name`, `engine_version`, `rule_id`, `rule_name`, `rule_uri`, `eval_result`, `eval_message`, `control_id`, `control_catalog_id`, `control_category`, `control_applicability`, `frameworks`, `requirements`, `requirement_id`, `plan_id`, `confidence`, `steps_executed`, `compliance_status`, `risk_level`, `remediation_action`, `remediation_status`, `remediation_desc`, `exception_id`, `exception_active`, `enrichment_status`, `attestation_ref`, `blob_ref`, and `collected_at`.

The handler **MUST** reject the request with `400 Bad Request` when per-row business validation fails (e.g. missing required fields for insert as defined in implementation), and **MUST** return a structured error payload listing row indices and reasons.

| ID | Scenario | Given | When | Then |
|:---|:---|:---|:---|:---|
| 1.1 | Full semconv row | A single object with all supported keys populated with valid enum/string/array values | Client sends `POST /api/evidence` with `Content-Type: application/json` | Server responds `201 Created` and persists one row with supplied fields; NULL-equivalent handled per column defaults. |
| 1.2 | Optional enrichment only | An object with required insert fields plus `requirement_id`, `plan_id`, `engine_name`, `confidence`, `compliance_status` | Client posts the array | Server persists with non-NULL `requirement_id` / `plan_id` where provided; OTel-originated and manual rows share the same `SELECT` shape for those columns. |
| 1.3 | Invalid enum | `eval_result` or `compliance_status` (or other enum-backed field) is not a legal value for ClickHouse | Client posts the array | Server responds `400` with a message identifying the field and value; no partial insert for that row. |
| 1.4 | Omitted semconv field | A valid minimal row omitting several nullable semconv fields | Client posts the array | Server persists; omitted fields store as NULL or table defaults; row still queryable with same `evidence` schema as collector-written rows. |

---

## REQ-2: `POST /api/evidence/upload` (CSV) validates columns

For CSV (and, when used, CSV part of multipart upload), the handler **MUST** verify presence of **required** header columns: `policy_id`, `eval_result`, `collected_at`.

The handler **SHOULD** include in the response a non-fatal **warning** when the header row omits the **recommended** column `requirement_id` (e.g. `warnings` array while still processing rows, or `207`-style document—exact HTTP pattern **MUST** be one consistent documented behavior; empty `requirement_id` cells are allowed if the column is present).

The handler **MUST** reject the upload (no row insert) when any **required** column is missing from the header.

The handler **MUST** reject or skip-with-error individual data rows with invalid `collected_at` (e.g. not RFC3339 or agreed alternate) and return structured per-line errors.

| ID | Scenario | Given | When | Then |
|:---|:---|:---|:---|:---|
| 2.1 | Valid minimal CSV | Header includes `policy_id`, `eval_result`, `collected_at` and optional semconv columns | User uploads file to `POST /api/evidence/upload` | At least one valid data row is inserted; response lists `inserted` count. |
| 2.2 | Missing `eval_result` column | Header lacks `eval_result` | User uploads | Server responds with error; message states missing required column; `inserted` is 0. |
| 2.3 | Missing `requirement_id` column | Required columns present; `requirement_id` absent from header | User uploads with otherwise valid rows | Server accepts insert **SHOULD** emit warning listing missing recommended column; rows store `requirement_id` per default/empty as schema allows. |
| 2.4 | Bad timestamp line | One line has unparseable `collected_at` | User uploads | That line is reported in `errors`; other valid lines **MAY** still insert (batch policy **MUST** be documented: all-or-nothing vs partial). |

---

## REQ-3: File artifacts use S3-compatible storage and set `blob_ref`

When the upload flow includes a file attachment (e.g. screenshot, PDF, log) associated with an evidence record, the gateway **MUST** upload the file bytes to configured S3-compatible object storage and **MUST** persist on the evidence row a non-NULL `blob_ref` for that record when a file is successfully stored.

The gateway **MUST** use `blob_ref` that references the object in a standard URI form (e.g. `s3://<bucket>/<key>`) so operators and tools can resolve location without a second lookup table.

If blob storage is misconfigured or the put fails, the handler **MUST** fail the operation for that record (or entire request per atomicity rules in `design.md`) and **MUST NOT** claim success for file retention.

| ID | Scenario | Given | When | Then |
|:---|:---|:---|:---|:---|
| 3.1 | File + row | A multipart request with evidence fields and a file part | User submits | Object exists at `blob_ref`; evidence row in ClickHouse has matching `blob_ref`. |
| 3.2 | No file | JSON-only or CSV-only ingest without attachment | User submits | `blob_ref` is NULL; insert otherwise succeeds. |
| 3.3 | Storage failure | S3-compatible API returns 5xx or network error | User submits with file | Request fails with `5xx` or `503` and no row with bogus `blob_ref`, or row rolled back per transaction design. |

---

## REQ-4: Ingestion parity (REST/CSV vs OTel)

For any field present on both the OTel ClickHouse write path and the manual ingest path, values **MUST** be stored in the same `evidence` columns with the same types; downstream `SELECT` queries (posture, inventory, agent SQL) **MUST** be able to use identical predicates and projections without branching on a synthetic "source" column unless one already exists for observability.

Manual ingest **MUST** set `enrichment_status` (and related defaults) so that NULL/absent user input does not silently pretend to be full OTel enrichment where the implementation defines otherwise.

| ID | Scenario | Given | When | Then |
|:---|:---|:---|:---|:---|
| 4.1 | Posture by `requirement_id` | One OTel row and one manual row for the same `requirement_id` and `policy_id` | An analyst runs `WHERE requirement_id = ?` | Both rows appear; counts and filters match semantics. |
| 4.2 | No `ingest_source` leak | N/A | N/A | Query that omits any ingest-channel filter still returns a unified evidence set. |
| 4.3 | Defaulted manual row | User omits `engine_name` on REST | Ingest completes | `engine_name` is NULL or default consistent with other non-OTel partial rows; not a distinct sentinel that breaks JOINs. |

---

## REQ-5: Schema — `blob_ref` column

The `evidence` table **MUST** include a nullable `blob_ref` column of type `Nullable(String)` (or equivalent) added via additive migration.

Existing rows **MUST** read as NULL for `blob_ref` after migration. New inserts without files **MUST** store NULL. Applications **MUST NOT** require `blob_ref` for non-file evidence.

| ID | Scenario | Given | When | Then |
|:---|:---|:---|:---|:---|
| 5.1 | Fresh migration | ClickHouse has prior `evidence` data without `blob_ref` | Migration runs | `ALTER` succeeds; `SELECT blob_ref` returns NULL for old rows. |
| 5.2 | New file row | A row is inserted with a file | Insert completes | `blob_ref` holds URI string. |
| 5.3 | Exporter/collect unchanged | OTel collector continues writing without `blob_ref` | Collector inserts | New rows have `blob_ref` NULL; no writer failure. |
