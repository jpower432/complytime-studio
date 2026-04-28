## ADDED Requirements

### Requirement: GET /api/requirements returns matrix rows scoped by policy and audit window

The Gateway SHALL expose `GET /api/requirements` that returns assessment requirements joined with evidence status aggregates, latest posture classification (from `evidence_assessments`), and staleness signals, filtered by `policy_id` and an audit time window. The response MUST be JSON and MUST include sufficient identifiers to join rows to policies, controls, and requirement IDs.

#### Scenario: Happy path — populated policy with assessments

- **GIVEN** ClickHouse contains `assessment_requirements` and linked `evidence` for `policy_id` P, and `evidence_assessments` rows for that evidence
- **WHEN** the client requests `GET /api/requirements?policy_id=P&audit_start=<t0>&audit_end=<t1>` with valid `t0` ≤ `t1`
- **THEN** the response returns HTTP 200 with a list of requirement rows including requirement text, evidence counts or presence, latest `collected_at` within the window where applicable, latest classification per covered evidence, and computed staleness where defined by product rules

#### Scenario: No evidence for a requirement

- **GIVEN** a requirement row exists in `assessment_requirements` for P but no `evidence` row links to that requirement in the window
- **WHEN** the client requests the same endpoint
- **THEN** the response SHALL include that requirement with zero or empty evidence status and a classification consistent with "Blind" or an explicitly documented unassessed state, without omitting the row

#### Scenario: Bad or missing query parameters

- **GIVEN** the client omits `policy_id` or supplies `audit_start` / `audit_end` in an invalid order or format
- **WHEN** the client calls `GET /api/requirements`
- **THEN** the Gateway SHALL return HTTP 400 with a body describing the validation error and SHALL NOT execute an unbounded scan

#### Scenario: Empty result set

- **GIVEN** `policy_id` references a policy with no matching assessment requirements, or a window that excludes all data
- **WHEN** the client requests `GET /api/requirements` with valid parameters
- **THEN** the response SHALL return HTTP 200 with an empty list (or paged empty page) and stable pagination metadata if pagination is used

### Requirement: GET /api/requirements/:id/evidence returns evidence for one requirement

The Gateway SHALL expose `GET /api/requirements/{id}/evidence` (where `{id}` identifies an assessment requirement in the context of the matrix API) that returns `evidence` rows linked to that requirement, including classification from `evidence_assessments` and timestamps. The handler MUST scope results by the same `policy_id` and audit window semantics as the list endpoint when those are provided as query parameters.

#### Scenario: Happy path — multiple evidence rows

- **GIVEN** several `evidence` rows reference the requirement and fall within the requested `policy_id` and audit window
- **WHEN** the client requests `GET /api/requirements/{id}/evidence?policy_id=P&audit_start=...&audit_end=...`
- **THEN** the response returns HTTP 200 with an ordered list of evidence records including `evidence_id`, source or target fields as stored, `collected_at`, and latest `classification` and `assessed_at` from `evidence_assessments FINAL` or equivalent

#### Scenario: Unknown requirement id

- **GIVEN** `{id}` does not match any assessment requirement the API can resolve for P
- **WHEN** the client requests the evidence endpoint
- **THEN** the Gateway SHALL return HTTP 404

#### Scenario: Stale or duplicate assessment history

- **GIVEN** multiple rows exist in `evidence_assessments` for the same `evidence_id` over time
- **WHEN** the client requests evidence for a requirement
- **THEN** each evidence row in the response SHALL surface the latest assessment per `evidence_id` (e.g. via `FINAL` or an aggregate) so the UI does not show duplicate classification timelines unless explicitly requested

### Requirement: Workbench requirement matrix view with filters

The Workbench SHALL provide a requirement matrix view that renders a filterable grid: framework or catalog lineage → control → assessment requirement → evidence and posture columns. The view MUST support filters for policy, control family (e.g. control group or category as exposed by the API), classification state, evidence staleness, and audit window, and MUST refetch or narrow data when filters change.

#### Scenario: User applies classification filter

- **GIVEN** the matrix is loaded for a policy
- **WHEN** the user selects a posture classification (e.g. Failing) and applies the filter
- **THEN** the grid shows only requirements whose aggregated or primary classification matches the selection per API contract, and the URL or client state reflects the filter for shareability where supported

#### Scenario: User changes audit window

- **GIVEN** `selectedTimeRange` or an equivalent workbench control updates the audit window
- **WHEN** the user applies a new end date or range
- **THEN** the matrix refetches from `GET /api/requirements` with the new window and updates counts and columns without a full page reload

#### Scenario: No policies imported

- **GIVEN** the user opens the matrix when no policy is selected or the policy list is empty
- **THEN** the view SHALL show an empty or instructional state and SHALL NOT throw an unhandled error

### Requirement: Drill-down from requirement row to evidence

The requirement matrix view SHALL let the user open or expand a requirement row to load linked evidence via `GET /api/requirements/{id}/evidence`, displaying sources, timestamps, and classifications. Loading and error states MUST be visible to the user.

#### Scenario: Expand row with evidence

- **GIVEN** a requirement row has linked evidence
- **WHEN** the user expands the row or follows the drill-down affordance
- **THEN** the workbench fetches the evidence endpoint and lists each row with classification and `collected_at`

#### Scenario: Network or server error on drill-down

- **GIVEN** the evidence request fails
- **WHEN** the user triggers drill-down
- **THEN** the UI shows a non-destructive error message and allows retry without leaving the matrix

#### Scenario: Large evidence list

- **GIVEN** a requirement has many evidence rows
- **WHEN** the user expands the row
- **THEN** the workbench either paginates the embedded list, truncates with "load more", or documents reliance on server-side limits so the main matrix remains responsive

### Requirement: PostureView links to the requirement matrix for drill-down

PostureView SHALL provide navigation to the requirement matrix with context (e.g. current `policy_id` and/or audit range) so users move from per-policy summary posture to requirement-level analysis without re-selecting context manually when those values are already known in PostureView.

#### Scenario: Link from a policy card

- **GIVEN** PostureView shows a policy summary for policy P
- **WHEN** the user activates the control that opens the requirement matrix
- **THEN** the workbench routes to the matrix with P pre-selected and a consistent audit window with the rest of the workbench (e.g. `selectedTimeRange`) where applicable

#### Scenario: Deep link or refresh

- **GIVEN** the user lands on the matrix with query parameters for policy and time range
- **WHEN** the page loads
- **THEN** filters initialize from those parameters and data loads without requiring PostureView to be visited first

#### Scenario: Viewer role

- **GIVEN** the user has read-only (viewer) access
- **WHEN** the user follows the link from PostureView
- **THEN** the matrix is readable and the Gateway returns 200 for GET endpoints; mutating actions remain forbidden per existing auth rules
