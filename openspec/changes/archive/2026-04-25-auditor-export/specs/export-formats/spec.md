# Export Formats (Delta)

**Capability:** `export-formats`  
**Change:** `auditor-export`  
**Key words:** RFC 2119 — “MUST”, “MUST NOT”, “SHOULD”, “MAY”.

## Scope

Structured exports (CSV, Excel, PDF) for auditor deliverables, scoped to one policy and one audit window, aligned with the requirement matrix and gateway REST patterns described in the change proposal and architecture docs.

---

## REQ-1: Scoping to policy and audit window

All export endpoints **MUST** accept parameters that identify exactly one `policy_id` and one audit window (`audit_start` and `audit_end` as instants, or an equivalent unambiguous date-time range). The exported content **MUST** reflect the compliance state for that policy only, using evidence and assessments whose timestamps fall within the audit window (per the same rules as the requirement matrix view).

### Scenarios

| # | GIVEN | WHEN | THEN |
|:--|:--|:--|:--|
| 1.1 | A valid `policy_id` and an audit window that overlaps stored evidence and assessments | The client calls any export endpoint with those parameters | The response body **MUST** include only rows derived from that policy and that window; other policies’ data **MUST NOT** appear. |
| 1.2 | A `policy_id` with no data in the window (empty period) | The client calls any export endpoint | The response **MUST** be a valid, empty or header-only artifact per format rules (e.g. CSV with headers, Excel with empty sheets or header rows as defined), and **MUST** still include generation metadata. |
| 1.3 | A malformed or missing `policy_id` or invalid audit range (e.g. `audit_start` after `audit_end`) | The client calls any export endpoint | The server **MUST** respond with `4xx` and **MUST NOT** return a successful file download. |

---

## REQ-2: `GET /api/export/csv`

The gateway **MUST** expose `GET /api/export/csv` that returns a CSV document whose rows represent requirement-level status for the scoped policy and audit window. Column names and order **MUST** match the requirement matrix view columns: requirement identifier, requirement text, control (or control identifier as shown in the matrix), evidence count, latest evidence date, classification, staleness, and any additional matrix columns the workbench displays for the same view (the CSV **MUST** stay in lockstep with the matrix API for that release).

### Scenarios

| # | GIVEN | WHEN | THEN |
|:--|:--|:--|:--|
| 2.1 | Requirements with mixed evidence: some with many rows, some with one | The client requests CSV with valid scoping | Each row **MUST** list the correct evidence count and latest evidence date; counts **MUST** match a direct query for that requirement in the window. |
| 2.2 | A requirement with zero evidence in the window | The client requests CSV | The row **MUST** appear with evidence count zero (or “0”) and null/empty latest evidence date as defined by the matrix; the row **MUST NOT** be omitted unless the matrix also omits it. |
| 2.3 | A large policy (many thousands of requirements) | The client requests CSV | The response **MUST** complete within documented limits (see design) or fail with an explicit error; partial CSV without completion **MUST NOT** be labeled as success. |

---

## REQ-3: `GET /api/export/excel`

The gateway **MUST** expose `GET /api/export/excel` that returns an Excel workbook (`.xlsx`) with at least the following **worksheet names and roles**:

1. **Executive Summary** — high-level posture for the policy and window (e.g. counts, key gaps) suitable for a client cover sheet.  
2. **Requirement Detail** — per-requirement status aligned with REQ-1 scoping.  
3. **Evidence Inventory** — evidence rows (or summary rows) tied to the policy and window with identifiers suitable for an auditor’s traceability.  
4. **Gap List** — requirements (or controls) not satisfied or missing evidence, per the same scoping.

### Scenarios

| # | GIVEN | WHEN | THEN |
|:--|:--|:--|:--|
| 3.1 | All requirements pass with evidence in window | The client requests Excel | All four sheets **MUST** be present; **Gap List** **MAY** be empty or state “no gaps” per product convention; other sheets **MUST** be populated consistently. |
| 3.2 | All requirements are gaps (no evidence in window) | The client requests Excel | **Gap List** and **Requirement Detail** **MUST** reflect full gap coverage; **Evidence Inventory** **MUST** be empty or header-only; **Executive Summary** **MUST** reflect the gap-heavy posture. |
| 3.3 | Evidence and assessments span many targets | The client requests Excel | **Evidence Inventory** **MUST** remain within scoping rules; the workbook **MUST** open in a mainstream Excel version without repair prompts for generated sizes within limits. |

---

## REQ-4: `GET /api/export/pdf`

The gateway **MUST** expose `GET /api/export/pdf` that returns a PDF document serving as a formatted compliance report for the same scoped policy and audit window, including a posture summary, requirement status summary or table, and gap analysis section consistent with the data underlying REQ-2 and REQ-3.

### Scenarios

| # | GIVEN | WHEN | THEN |
|:--|:--|:--|:--|
| 4.1 | Normal data with a mix of compliant and gapped requirements | The client requests PDF | The PDF **MUST** include readable sections for posture summary, requirement table or summary, and gaps; page structure **MAY** paginate as needed. |
| 4.2 | Scoping window with no evidence | The client requests PDF | The PDF **MUST** state the empty or unknown posture explicitly and **MUST** still include generation metadata. |
| 4.3 | A very large row set (stress case) | The client requests PDF | The server **MUST** either complete within timeout limits with acceptable truncation **documented in design** or return `4xx/5xx` with a clear message; silent truncation **MUST NOT** occur without documentation. |

---

## REQ-5: Generation metadata

Every successful export (CSV, Excel, PDF) **MUST** embed or precede the payload with **generation metadata** that includes: generation timestamp, policy version (or policy identifier and version string as stored), audit window (start and end), and generator identity (e.g. product name and version string).  
- For CSV, metadata **MUST** appear as a leading comment block, dedicated preamble rows, or a companion row convention **documented in the API** and mirrored in the workbench.  
- For Excel and PDF, metadata **MUST** appear on the **Executive Summary** (or first page) in human-readable form.

### Scenarios

| # | GIVEN | WHEN | THEN |
|:--|:--|:--|
| 5.1 | Any successful export | The client inspects the artifact | Generator and timestamps **MUST** be present and **MUST** match the request’s policy and window. |
| 5.2 | Policy was re-imported; version string updated | The client exports after the update | The metadata **MUST** reflect the current stored policy version for `policy_id`. |
| 5.3 | Re-export the same window seconds later | The client exports twice | Generation timestamp **MUST** differ (or at minimum reflect the second run’s time). |

---

## REQ-6: Optional agent-produced narrative (Excel and PDF)

Excel and PDF exports **MAY** include agent-produced narrative sections (e.g. executive summary prose, gap commentary) when such content is available in the system (e.g. linked `AuditLog` or summary field for the policy and window). If narrative is unavailable, the export **MUST** still succeed using structured data only.  
CSV **MUST NOT** be required to embed long narrative; if short classification text is part of matrix columns, it **MUST** remain columnar per REQ-2.

### Scenarios

| # | GIVEN | WHEN | THEN |
|:--|:--|:--|
| 6.1 | An `AuditLog` (or stored summary) exists with narrative for the same policy and window | The client requests Excel or PDF | Narrative **MAY** appear in **Executive Summary** and/or a dedicated section; location **MUST** be defined in design. |
| 6.2 | No agent narrative exists | The client requests Excel or PDF | The files **MUST** generate without error; template sections **MAY** show placeholders or remain minimal. |
| 6.3 | Narrative exists but conflicts with current evidence counts | The client requests Excel | Structured sheets **MUST** reflect ClickHouse truth; narrative **MUST** be labeled as agent-generated or **SHOULD** be omitted if product policy forbids inconsistency. |

---

## REQ-7: `Content-Type` and download behavior

Successful CSV responses **MUST** use an appropriate `Content-Type` (e.g. `text/csv; charset=utf-8`).  
Successful Excel and PDF responses **MUST** use `application/vnd.openxmlformats-officedocument.spreadsheetml.sheet` and `application/pdf` respectively.  
Responses **SHOULD** include `Content-Disposition` with an attachment filename following project naming conventions (see `design.md`).

### Scenarios

| # | GIVEN | WHEN | THEN |
|:--|:--|:--|
| 7.1 | Any format | A successful export | `Content-Disposition` **SHOULD** suggest a filename containing policy id and date range. |
| 7.2 | Client omits accept headers | The server returns the file | Body **MUST** still match the requested route’s format. |

---

## Consistency

Export row counts and key aggregates **SHOULD** match between CSV, Excel **Requirement Detail**, and PDF summary tables for the same request parameters, modulo PDF truncation policy for huge datasets.
