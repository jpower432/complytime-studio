# Design: `auditor-export`

## 1. Excel generation library

**Decision:** Use **[excelize](https://github.com/xuri/excelize)** (Go) for `.xlsx` generation.

| Factor | Rationale |
|:--|:--|
| Ecosystem | Mature, Apache 2.0, widely used; fits the Go gateway. |
| Features | Multiple sheets, styling, and column sizing sufficient for the four target sheets. |
| Operations | No separate runtime (unlike a headless browser) — simpler in the existing gateway binary. |

**Alternatives considered:** `tealeg/xlsx` (less active feature set for styling); server-side OpenXML templates (more moving parts for v1).

---

## 2. PDF generation: server vs client

**Decision:** **Server-rendered PDF** in the gateway for v1.

| Factor | Rationale |
|:--|:--|
| Provenance | One implementation path: same query layer as CSV/Excel, auditable in CI. |
| UX | User gets a file immediately; no browser print differences. |
| Security | No reliance on client-side user agents for the canonical deliverable. |

**Pattern:** Build report content from structured data in Go, then render via a small Go-friendly PDF layer (e.g. **gofpdf**, **gopdf**, or **pdfcpu**-based layout — exact crate chosen at implementation; criteria: stable API, FLOSS license, no CGO if possible for simpler builds). Optional HTML **→** PDF (e.g. `chromedp` or `wkhtmltopdf`) is **out of scope for v1** unless layout complexity forces it: adding a headless stack increases Helm image size and flakiness.

**Client print:** The workbench **MAY** add “Print to PDF” later; it is not the normative export for auditors.

---

## 3. Agent narrative integration

**Decision:** Treat narrative as **optional** text sourced from existing persisted audit artifacts, keyed by scoping.

| Source | Use |
|:--|:--|
| `audit_logs` | The table already has a `summary` column (`internal/clickhouse/client.go`). If an `AuditLog` row exists for the same `policy_id` with `audit_start` / `audit_end` matching the export window (or a selected `audit_id` query param), that summary **MAY** populate **Executive Summary** prose and “Gap commentary” blocks in Excel/PDF. |
| A2A / agent | No new transport: exports read **only** what the gateway has already persisted. Future: explicit `narrative_audit_id` query param to pick one log when several overlap. |

**Rule:** Structured sheets and tables are **source of truth** from ClickHouse joins; agent text is **annotative** and labeled as such if included.

---

## 4. ClickHouse access pattern

**Decision:** **Compose queries** aligned with the requirement matrix API, reusing the same logical joins the matrix uses — **not** a single monolithic string for all formats. Shared Go functions **SHOULD** build: (1) requirement-level rows, (2) evidence inventory rows, (3) gap list derivation (requirements failing “has evidence in window” or equivalent rule).

| Approach | Rationale |
|:--|:--|
| Reuse matrix semantics | Prevents CSV columns drifting from the grid. |
| Multiple queries acceptable | Clarity and testability; `evidence` is large — narrow projections per sheet. |
| Views | If `unified_compliance_state` / `policy_posture` (or successors) exist, handlers **SHOULD** query them for aggregates; fall back to explicit joins on `assessment_requirements`, `evidence`, `evidence_assessments` as in the requirement-matrix change. |

**No schema change** for this change (per proposal): export logic is read-only on existing tables.

---

## 5. File size, streaming, and timeouts

| Concern | Mitigation |
|:--|:--|
| **Memory** | Stream CSV; for Excel, stream rows into excelize with bounded in-memory book or chunk where the library allows; if memory-bound, cap row counts and return `413` or `500` with message. |
| **Gateway timeout** | Document a maximum row count and/or `export_deadline` env (e.g. 60s default); return `504` on overrun. |
| **Large PDF** | Summarize in PDF (top N gaps + full counts) if row count exceeds threshold; **document** truncation; never silently drop without indicator in the PDF. |

**CSV** is the preferred path for “full dump”; Excel/PDF may apply limits first.

---

## 6. HTTP headers and filenames

| Header | Value |
|:--|:--|
| `Content-Type` | `text/csv; charset=utf-8` \| `application/vnd.openxmlformats-officedocument.spreadsheetml.sheet` \| `application/pdf` |
| `Content-Disposition` | `attachment; filename="<safe-name>.<ext>"` |

**Filename convention (ASCII, filesystem-safe):**

`complytime-export_<policyIdShort>_<YYYY-MM-DD>_<YYYYMMDD-HHMM>_<format>.<ext>`

- `policyIdShort`: truncated UUID or slug (max ~16 chars) to avoid path length issues.  
- Date segment: audit window end date (or start–end as two segments if product prefers).  
- Sanitize: strip characters outside `[A-Za-z0-9._-]`.

**Cache:** `Cache-Control: no-store` for all export responses.

---

## 7. Authentication and authorization

Exports follow existing gateway rules (`docs/design/architecture.md`): when OAuth is enabled, only authenticated users; mutating admin-only routes do not apply to GET export, but read scope **MUST** respect the same read access as the requirement matrix (viewers: read).

---

## 8. Open questions (non-blocking for spec)

- Whether export accepts `audit_id` in addition to raw date range to pin one `AuditLog` for narrative.  
- Exact PDF layout library after spike (pure-Go vs HTML pipeline).
