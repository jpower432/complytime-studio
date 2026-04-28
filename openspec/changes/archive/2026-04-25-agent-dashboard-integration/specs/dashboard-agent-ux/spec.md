# Delta Spec: Dashboard and Agent UX

## MODIFIED Requirements

### Requirement: PostureView uses gateway REST aggregates

PostureView MUST NOT render agent-produced summary cards as the source of compliance posture. PostureView SHALL load posture data by calling the Studio gateway REST API, which reads the ClickHouse `policy_posture` view. The workbench browser MUST NOT open direct ClickHouse connections for posture.

#### Scenario: Initial load fetches REST aggregates
- **GIVEN** the analyst opens PostureView
- **WHEN** the view mounts and the session is authenticated (or auth is disabled for development)
- **THEN** the workbench issues `GET /api/posture` (with applicable query parameters for filter scope)
- **AND** the response body SHALL contain JSON rows derived from `policy_posture` (not from chat or agent session state)

#### Scenario: Stale or empty agent content is not shown as posture
- **GIVEN** a prior session produced an AuditLog or chat summary with policy-level narrative
- **WHEN** PostureView renders
- **THEN** that narrative SHALL NOT appear as the primary posture breakdown in place of `policy_posture` aggregates
- **AND** if the REST call fails, the UI SHALL show an error state and SHALL NOT fall back to cached agent text as authoritative posture

#### Scenario: Consistency with other dashboard contracts
- **GIVEN** `policy_posture` is the documented aggregate for policy/target summaries
- **WHEN** PostureView displays counts or status per policy and target
- **THEN** those values SHALL match the same logical rows the gateway exposes via `GET /api/posture` for the same scope

---

### Requirement: `GET /api/posture` exposes `policy_posture` as JSON

The gateway SHALL implement `GET /api/posture` and SHALL return a JSON document representing aggregated rows from the ClickHouse `policy_posture` view for the request scope. The response schema SHALL be stable and versioned in implementation notes; clients SHALL treat the endpoint as the single source of truth for dashboard posture until superseded by an explicit spec change.

#### Scenario: Success returns aggregate rows
- **GIVEN** ClickHouse contains `policy_posture` and the gateway has read access
- **WHEN** a client calls `GET /api/posture` with valid auth (when auth is enabled)
- **THEN** the HTTP status SHALL be `200`
- **AND** the body SHALL be JSON with an array (or top-level object containing an array) of aggregate records aligned to the view’s columns

#### Scenario: Unauthenticated or unauthorized access is rejected when auth is on
- **GIVEN** Google OAuth is enabled and the request lacks a valid session
- **WHEN** the client calls `GET /api/posture`
- **THEN** the gateway SHALL return `401` (or the same behavior as other `/api` GET routes for unauthenticated users)

#### Scenario: Data store error surfaces clearly
- **GIVEN** ClickHouse is unavailable or the query on `policy_posture` fails
- **WHEN** the handler runs the read
- **THEN** the response SHALL be an error status (e.g. `502` or `500`) with a non-empty error payload suitable for logging
- **AND** the response MUST NOT return fabricated posture rows

---

### Requirement: Chat overlay accepts workbench view context

The ChatAssistant (chat overlay) MUST accept pre-populated structured context from the active workbench view, including at minimum: current route/view name, selected `policy_id`, and when applicable control and evidence filter dimensions. That context SHALL be merged into the injected payload sent to the agent on new tasks (or first message of a task) using the same general mechanism as `buildInjectedContext` (structured serialization, not a cold-start free-text only prompt).

#### Scenario: Posture view injects policy and time scope
- **GIVEN** the analyst is on PostureView with a selected policy and optional time range
- **WHEN** they open the chat overlay and send a message on a new task
- **THEN** the A2A request SHALL include injected text containing `view`, `policy_id`, and time range fields where present
- **AND** the agent SHALL be able to reason from that structure without the user re-stating the policy in natural language

#### Scenario: Evidence view injects filter dimensions
- **GIVEN** the analyst applies filters (e.g. policy, target, control, date range) on the evidence browser
- **WHEN** they send a message with the overlay open
- **THEN** those filter keys and values SHALL appear in the injected context record
- **AND** a message sent without changing filters SHALL still carry the current filter set on first send of a new task

#### Scenario: Sticky notes remain additive
- **GIVEN** the user has sticky notes enabled
- **WHEN** view context and sticky notes are both present
- **THEN** the assembled injection SHALL include both the dashboard context JSON and the `<sticky-notes>` block as today
- **AND** the order of assembly SHALL not drop either source

---

### Requirement: Canned query affordances in the workbench

The workbench SHALL expose one-click (or one-tap) actions in the chat overlay for these workflows: "Run posture check", "Generate AuditLog", and "Summarize gaps". Selecting a canned action SHALL populate the user message field (or send a predetermined user message) that routes the agent to the correct workflow per the assistant prompt, without requiring the user to type the full intent.

#### Scenario: Run posture check
- **GIVEN** the chat overlay is open
- **WHEN** the user activates "Run posture check"
- **THEN** the client SHALL send a user message (or equivalent) that unambiguously triggers the Posture Check Workflow per the assistant prompt
- **AND** the injected view context SHALL be included for the same send path as a manual message

#### Scenario: Generate AuditLog
- **GIVEN** the chat overlay is open
- **WHEN** the user activates "Generate AuditLog"
- **THEN** the client SHALL send a user message that routes to Audit Production Workflow
- **AND** policy and audit window from context SHALL be used when present so the agent does not need to re-ask unless required fields are missing

#### Scenario: Summarize gaps
- **GIVEN** the chat overlay is open
- **WHEN** the user activates "Summarize gaps"
- **THEN** the client SHALL send a user message that requests gap-oriented synthesis over the current scope
- **AND** the user MAY edit the message before send if the UI supports prefilling the input

---

## ADDED Requirements

### Requirement: Agent artifacts refresh dashboard views via SSE and persistence

When the agent produces `AuditLog` or `EvidenceAssessment` artifacts, the workbench’s dashboard views that surface those entity types SHALL update in near real time. Updates SHALL use the existing A2A SSE stream for artifact events combined with gateway auto-persistence to ClickHouse. Views SHALL not require a full page reload to show newly persisted rows when the user remains in-session.

#### Scenario: Auto-persisted AuditLog appears in history
- **GIVEN** auto-persistence is enabled and the agent streams an AuditLog artifact
- **WHEN** the gateway persists the row to `audit_logs`
- **THEN** AuditHistoryView (or equivalent) SHALL reflect the new entry after the next data refresh cycle triggered from the stream or app-level invalidation
- **AND** the user SHALL see consistency between the chat artifact card and the history list

#### Scenario: EvidenceAssessment updates evidence-oriented surfaces
- **GIVEN** the agent emits an `EvidenceAssessment` artifact and the gateway persists related assessment data
- **WHEN** Posture or evidence surfaces depend on that assessment linkage
- **THEN** the workbench SHALL refresh or invalidate queries so aggregates do not show stale "unassessed" state indefinitely without manual reload

#### Scenario: No duplicate persistence from the browser for streamed artifacts
- **GIVEN** the gateway has already auto-persisted a streamed Gemara artifact
- **WHEN** the user views the dashboard
- **THEN** the client SHALL NOT POST duplicate rows solely because SSE delivered the artifact; optional manual "Save" paths remain governed by existing admin and UX rules

---

### Requirement: PostureView links to requirement matrix drill-down

PostureView SHALL provide navigation to the requirement matrix view (per-policy or scoped drill-down) so the analyst can move from aggregate posture to per-requirement status. The link or button SHALL pass sufficient route state (e.g. `policy_id` and scope parameters) to align the matrix with the same selection on PostureView.

#### Scenario: From policy row to matrix
- **GIVEN** PostureView shows a row or card for a policy
- **WHEN** the user chooses the requirement-matrix drill-down control for that policy
- **THEN** the router SHALL open the requirement matrix view
- **AND** the matrix SHALL be scoped to that policy (or show an explicit empty state if no requirements exist)

#### Scenario: No matrix route yet (incremental delivery)
- **GIVEN** the requirement matrix route is behind a feature flag or not yet merged
- **WHEN** the link is not available
- **THEN** PostureView MAY hide the control or show disabled state with a short explanation per release notes
- **AND** the spec remains satisfied once the matrix route ships with the described behavior

#### Scenario: Deep link preserves context for agent
- **GIVEN** the user navigates from PostureView to the matrix
- **WHEN** they open the chat overlay
- **THEN** `currentView` and `policy_id` in injected context SHALL reflect the matrix view and selection so handoff remains coherent

---

## REMOVED Requirements

### Requirement: Retire agent-as-primary posture presentation

The pattern where PostureView treats agent-generated summary cards as the authoritative posture breakdown SHALL be removed from supported UX. The agent MAY still discuss or explain posture, but the grid SHALL be driven by `policy_posture` via `GET /api/posture` as specified above.

#### Scenario: No regression to chat-only posture for routine checks
- **GIVEN** an analyst asks "what is our posture?" in the UI sense (glanceable dashboard)
- **WHEN** they use PostureView
- **THEN** the answer SHALL come from the REST-backed aggregates, not from requiring a chat turn first

#### Scenario: Documentation alignment
- **GIVEN** prior copy described the agent as replacing dashboards
- **WHEN** this change is complete
- **THEN** in-app help and OpenSpec parent proposal SHALL align with dashboard-first posture and contextual chat

#### Scenario: Agent remains optional for reading posture
- **GIVEN** the agent pod is down
- **WHEN** the analyst opens PostureView
- **THEN** posture aggregates SHALL still load from `GET /api/posture` subject only to ClickHouse and gateway health
