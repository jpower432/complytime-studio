## Why

Auditors deliver compliance reports as Excel workbooks and PDFs. Their clients expect structured documents showing requirement status, evidence references, coverage gaps, and remediation recommendations — not chat transcripts or raw YAML.

Studio produces AuditLog artifacts and surfaces posture data, but there is no export path. An analyst who completes an audit in Studio must manually copy results into a spreadsheet to produce the deliverable. This negates the efficiency gain of using Studio in the first place.

The [cloud-native posture correction](../../docs/decisions/cloud-native-posture-correction.md) establishes that structured views with export capability are the primary analyst interface.

## What Changes

- **CSV export from requirement matrix**: One-click export of the requirement matrix view as CSV. Columns match the grid: requirement ID, text, control, evidence count, latest evidence date, classification, staleness.
- **Excel workbook export**: Structured workbook with sheets for executive summary, requirement detail, evidence inventory, and gap list. Formatted for direct use as an auditor deliverable.
- **PDF report generation**: Rendered from the same data as the Excel export. Includes posture summary, requirement status table, and gap analysis.
- **Per-audit-period scoping**: All exports are scoped to a policy and audit window. The export reflects the point-in-time compliance state for that period.
- **Agent-assisted drafting**: The agent can produce narrative sections (executive summary, gap analysis commentary) that are included in the export alongside the structured data.

## Capabilities

### New Capabilities
- `csv-export`: One-click CSV download from the requirement matrix view
- `excel-export`: Structured Excel workbook with multiple sheets for auditor delivery
- `pdf-export`: Formatted PDF compliance report
- `export-scoping`: All exports scoped to policy + audit window for point-in-time reporting

### Modified Capabilities
- `requirement-matrix-view`: Export buttons added to the view toolbar
- `audit-production-workflow`: Agent can produce narrative sections for inclusion in exports

## Impact

- **Gateway**: New REST endpoints (`GET /api/export/csv`, `GET /api/export/excel`, `GET /api/export/pdf`) with query parameters for policy, audit window, and optional narrative sections
- **Workbench**: Export buttons in requirement matrix toolbar. Format picker (CSV/Excel/PDF).
- **Dependencies**: Go library for Excel generation (e.g., excelize). PDF generation via HTML-to-PDF or template rendering.
- **Agent**: Optional — agent produces narrative text via existing A2A flow; gateway includes it in export if available
- **ClickHouse**: No schema changes — exports query existing tables and views

## Constitution Alignment

### I. Autonomous Collaboration

**Assessment**: PASS

Exports are self-contained artifacts. They include all context needed for the recipient (auditor, client) without requiring access to Studio.

### II. Composability First

**Assessment**: PASS

CSV, Excel, and PDF are independent export formats. Each is usable standalone. Agent-produced narrative sections are optional — exports work without them.

### III. Observable Quality

**Assessment**: PASS

Every row in the export traces to specific evidence and assessment records with timestamps and provenance. Export metadata includes generation date, policy version, and audit window.

### IV. Testability

**Assessment**: PASS

Exports testable with seeded data — verify row counts, column completeness, and scoping accuracy. PDF/Excel output verifiable by parsing the generated files in integration tests.
