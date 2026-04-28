## Why

Compliance analysts track readiness at the requirement level, not the policy level. The current PostureView shows per-policy summary cards derived from agent-produced AuditLog summaries. There is no view that answers "for SOC 2 CC6.1, which requirements have evidence, which are gaps, and what's stale?" without asking the agent.

Analysts replacing Hyperproof need a grid they can open, filter, scan, and export — not a conversation. The [cloud-native posture correction](../../docs/decisions/cloud-native-posture-correction.md) retires "agent replaces dashboards" in favor of structured views augmented by the agent.

## What Changes

- **Requirement matrix view** in the workbench: a filterable grid showing Framework → Control → Requirement → Evidence status. Rows are assessment requirements (from `assessment_requirements` table). Columns include requirement text, evidence count, latest evidence date, posture classification (from `evidence_assessments`), and staleness.
- **REST endpoints** to serve the matrix data: pre-joined queries over `assessment_requirements`, `evidence`, and `evidence_assessments` scoped by policy and audit window.
- **Drill-down**: clicking a requirement row expands to show linked evidence rows with their classifications, sources, and timestamps.
- **Filtering**: by policy, control family, classification state (Healthy/Failing/Blind/etc.), evidence staleness, and audit window.

## Capabilities

### New Capabilities
- `requirement-matrix-view`: Workbench grid component showing requirement-level compliance status with filtering and drill-down
- `requirement-matrix-api`: REST endpoints serving pre-joined requirement + evidence + assessment data

### Modified Capabilities
- `posture-view`: PostureView links to the requirement matrix for drill-down instead of relying solely on agent-produced summaries

## Impact

- **Workbench**: New view component with route, added to sidebar navigation
- **Gateway**: New REST endpoints (`GET /api/requirements`, `GET /api/requirements/:id/evidence`) with ClickHouse queries joining `assessment_requirements`, `evidence`, and `evidence_assessments`
- **ClickHouse**: No schema changes — queries use existing tables and the `unified_compliance_state` / `policy_posture` views
- **Agent**: No changes — agent queries the same tables independently

## Constitution Alignment

### I. Autonomous Collaboration

**Assessment**: PASS

The matrix view consumes the same ClickHouse data the agent uses. Both operate independently — the view does not depend on agent output, and the agent does not depend on the view.

### II. Composability First

**Assessment**: PASS

REST endpoints return JSON. The view is a standalone workbench route. No mandatory coupling to other views or agent workflows.

### III. Observable Quality

**Assessment**: PASS

Every cell in the matrix traces to a specific `evidence_id` and `evidence_assessments` classification with timestamps and provenance (`assessed_by`).

### IV. Testability

**Assessment**: PASS

REST endpoints testable with seeded ClickHouse data. View component testable with mock API responses. Classification states are deterministic from data.
