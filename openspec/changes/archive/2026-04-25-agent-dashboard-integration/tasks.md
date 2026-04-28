# Tasks: Agent-Dashboard Integration

## Posture REST endpoint

- [x] Add `GET /api/posture` handler in the gateway; query ClickHouse via `internal/store` with same auth as other read APIs
- [x] Define JSON response shape and document in handler comments; align column names to the view
- [x] Return appropriate HTTP errors when ClickHouse is unavailable (no empty fabricated rows)
- [x] Add unit or integration test with testcontainers/fake store per repo conventions — `internal/store/posture_handler_test.go` (fake `PostureStore`, no testcontainers in repo yet)

## PostureView refactor

- [x] Replace agent-summary card data path with `apiFetch`/`GET /api/posture` on mount and on scope change
- [x] Map JSON to existing layout components; add loading and error states
- [x] Add navigation control from PostureView to requirement matrix with `policy_id`
- [x] Remove dead code that treated chat or ad-hoc agent text as the primary posture source

## Context injection

- [x] Extend `buildDashboardContext` to include policy, time range, and view-specific fields (control, requirement, evidence filters) from shared app state
- [x] Wire requirement matrix to set new state when selection changes (`selectedRequirementId`)
- [x] Keep `buildInjectedContext` as the single assembly point; ensure sticky notes behavior unchanged
- [x] On `streamReply`, confirm documented behavior: no double-inject on follow-up turns — documented in `workbench/src/components/chat-assistant.tsx` (see `send` / `streamReply`)

## Canned queries

- [x] Add three overlay buttons: "Run posture check", "Generate AuditLog", "Summarize gaps" with pre-filled user text
- [x] Reuse the same `send` path as manual input so injection runs on new tasks
- [x] (Optional) Surface labels/disabled state from config for ops-controlled rollout — **Deferred:** no workbench config surface in this change; follow up if product needs feature flags

## Real-time updates

- [x] In `onArtifact`, detect relevant artifact names and trigger debounced refetch via `invalidateViews()`
- [x] PostureView subscribes to `viewInvalidation` signal for auto-refetch
- [x] EvidenceView subscribes to `viewInvalidation` signal for auto-refetch
- [x] For Audit History, wire `viewInvalidation` to refetch `GET /api/audit-logs`
- [x] Document any limitation: same-session vs cross-tab; focus refetch for out-of-band updates — `workbench/src/app.tsx` (`viewInvalidation` comment)

## Testing

- [x] API test: `GET /api/posture` returns seeded aggregates; 401 when auth required and unauthenticated — `TestListPostureHandler`, `TestListPostureHandler_AuthMiddleware` in `internal/store/posture_handler_test.go`
- [x] Workbench test: open PostureView -> network shows `/api/posture` -> no agent request required for table body — **Manual verification complete** (no component test harness in repo)
- [x] Manual or automated check: canned button sends expected substring — **Manual verification complete** (buttons set input to strings containing e.g. `Run a posture check`, `AuditLog`, `compliance gaps`)
- [x] Regression: chat still streams; auto-persisted artifact still appears in history after refetch — **Manual verification complete** (no E2E harness)
