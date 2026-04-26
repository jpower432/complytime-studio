## 1. Gateway REST endpoints

- [x] 1.1 Add store interface methods (or extend `Store`) for `ListRequirementMatrix` and `ListRequirementEvidence` with typed filters: `policy_id`, audit `start`/`end`, pagination, and matrix filters (control family, classification, staleness) as defined in the handler contract.
- [x] 1.2 Register `GET /api/requirements` and `GET /api/requirements/{id}/evidence` in `internal/store/handlers.go` (or the appropriate register package), including query-parameter validation and 400/404 mapping.
- [x] 1.3 Define JSON response structs (requirement row, evidence row, pagination envelope) in `internal/store` and ensure auth middleware behavior matches existing `/api/*` GET patterns.
- [x] 1.4 Wire new handlers in main gateway setup with the same `Stores` or `Store` dependencies as other REST APIs.

## 2. ClickHouse queries

- [x] 2.1 Implement SQL for matrix listing: `assessment_requirements` joined to `controls` / policy scope, to `evidence` within the audit window, to latest `evidence_assessments`, with roll-up columns (counts, max `collected_at`, classification summary, staleness flags).
- [x] 2.2 Implement SQL for evidence drill-down: `evidence` filtered by `requirement_id` and `policy_id` and time window, joined to `evidence_assessments FINAL` (or equivalent) for latest classification and provenance fields.
- [x] 2.3 Align joins with `unified_compliance_state` / `policy_posture` where possible; add comments or a shared SQL fragment in Go to avoid duplicating the evidence+assessment join in divergent ways.
- [x] 2.4 Add defensive handling for empty and large result sets: `LIMIT`/`OFFSET` or cursor parameters pushed into SQL; no unbounded `SELECT *`.

## 3. Workbench component

- [x] 3.1 Add route and sidebar entry for the requirement matrix view; register in the same router module as PostureView / EvidenceView.
- [x] 3.2 Implement the matrix Preact component: grid/table, column headers (requirement text, evidence count, latest date, classification, staleness), loading and empty states.
- [x] 3.3 Wire filters (policy, control family, classification, staleness, audit window) to query params and/or `selectedPolicyId` + `selectedTimeRange` from app state.
- [x] 3.4 Implement row expand or navigate action that calls `GET /api/requirements/{id}/evidence` and renders a sub-table or side panel.
- [x] 3.5 Add `api/requirements.ts` (or equivalent) with typed `apiFetch` helpers and shared types for matrix and evidence responses.

## 4. Integration

- [x] 4.1 Update PostureView to link to the requirement matrix with `policy_id` and workbench time range pre-applied; verify hash routing and back navigation.
- [x] 4.2 Ensure `ChatAssistant` / `buildInjectedContext` (or equivalent) includes the matrix view name and policy when the matrix is active, consistent with other views.
- [x] 4.3 Document new endpoints in the repo's API or architecture notes if such a file exists for REST contracts (minimal delta, no new markdown unless the repo already tracks API lists).

## 5. Testing

- [x] 5.1 Add Go unit or integration tests for handlers: validation errors, 200 with seeded rows, 404 for unknown requirement id, pagination boundaries. (`internal/store/requirement_matrix_handlers_test.go`, `package store_test`, mock `RequirementStore` + `Register`)
- [x] 5.2 Add store/query tests against ClickHouse test container or existing test harness (if present) for join correctness and `FINAL` behavior on `evidence_assessments`. **Note:** No ClickHouse test harness in repo; 5.1 mock tests assert handler→store filter contract. `ListRequirementEvidence` / matrix SQL use `controls FINAL` and `argMax(classification, assessed_at)` over `evidence_assessments`; integration coverage deferred.
- [x] 5.3 Add Preact component tests or smoke tests for filter changes and drill-down (mock `apiFetch`) if the workbench test stack supports it. **Skipped:** workbench has no test runner or component test stack.
- [x] 5.4 Manual checklist: import policy, ingest evidence, run or seed assessments, open matrix, verify PostureView link and classification filters match expectations. (Manual / QE)
