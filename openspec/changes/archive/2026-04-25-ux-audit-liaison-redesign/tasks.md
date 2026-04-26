## 1. Navigation Restructure (Frontend)

- [x] 1.1 Update `View` type in `app.tsx` — remove `"audit-history"` and `"draft-review"`, add `"inbox"` and `"posture-detail"`
- [x] 1.2 Update `sidebar.tsx` — remove Audit History and Review nav items, add Inbox with unread badge
- [x] 1.3 Add `parseNestedHash` to support `#/posture/{policy_id}?tab=X` routes
- [x] 1.4 Add legacy redirect: `#/audit-history` → `#/posture`, `#/draft-review` → `#/inbox`
- [x] 1.5 Update `App` component routing to render `PolicyDetailView` for `posture-detail` and `InboxView` for `inbox`

## 2. Posture Drill-Down

- [x] 2.1 Create `PolicyDetailView` component with breadcrumb (Posture > [Policy Title])
- [x] 2.2 Implement tabbed layout: Requirements, Evidence, History
- [x] 2.3 Move `AuditHistoryView` into the History tab — remove policy selector (use parent context)
- [x] 2.4 Move requirement matrix into Requirements tab — pre-fill policy from parent
- [x] 2.5 Create Evidence tab showing evidence records filtered by active policy
- [x] 2.6 Update `PostureCard` onClick to navigate to `#/posture/{policy_id}` instead of linking to separate views
- [x] 2.7 Remove standalone `audit-history-view.tsx` import from `app.tsx`

## 3. Inventory Cards

- [x] 3.1 Extend `GET /api/posture` response with `target_count`, `control_count`, `latest_evidence_at`, `owner` fields
- [x] 3.2 Update `ListPosture` store method and SQL query to join evidence/controls/policy contacts
- [x] 3.3 Update `PostureRow` struct with new fields
- [x] 3.4 Update `PostureCard` UI to display target count, control count, evidence freshness, and owner
- [x] 3.5 Add relative time formatting utility (e.g., "2h ago", "3 days ago")

## 4. Inbox View

- [x] 4.1 Create `notifications` ClickHouse table: `notification_id, type, policy_id, payload, read, created_at` with TTL
- [x] 4.2 Add DDL to `internal/clickhouse/client.go` schema migration
- [x] 4.3 Create `NotificationStore` interface: `InsertNotification`, `ListNotifications`, `MarkRead`, `UnreadCount`
- [x] 4.4 Implement `NotificationStore` on `Store`
- [x] 4.5 Register `GET /api/notifications`, `GET /api/notifications/unread-count`, `PATCH /api/notifications/{id}/read` handlers
- [x] 4.6 Create `InboxView` component — list notifications + draft audit logs, sorted by creation time
- [x] 4.7 Move draft review functionality (type overrides, notes, promote) into inbox detail panel
- [x] 4.8 Implement sidebar badge polling (30s interval) for unread count
- [x] 4.9 Remove standalone `draft-review-view.tsx` import from `app.tsx`

## 5. NATS Event Bus

- [x] 5.1 Add `nats.go` dependency: `go get github.com/nats-io/nats.go@latest`
- [x] 5.2 Create `internal/events/nats.go` — connect, publish, subscribe helpers
- [x] 5.3 Update `cmd/ingest/main.go` — publish `studio.evidence.{policy_id}` after evidence insert
- [x] 5.4 Update `cmd/gateway/main.go` — subscribe to `studio.evidence.*` on startup
- [x] 5.5 Implement 30-second per-policy debounce in gateway subscriber
- [x] 5.6 Implement in-flight tracking — skip duplicate posture checks per policy
- [x] 5.7 Fire-and-forget publish in ingest — log warning on NATS failure, never block insert
- [x] 5.8 Add NATS Deployment, Service, NetworkPolicy to Helm chart (gated by `nats.enabled`)
- [x] 5.9 Add `NATS_URL` env var to gateway and ingest templates
- [x] 5.10 Add `nats` section to `values.yaml` with `enabled: false`, `image`, `resources`

## 6. Event-Triggered Posture Check

- [x] 6.1 Create gateway handler: on debounced evidence event, POST to agent A2A with posture-check context
- [x] 6.2 Update `posture-check` skill to accept event-triggered input (`policy_id` + `record_count`)
- [x] 6.3 Return lightweight posture delta (previous/current pass rate, new finding count)
- [x] 6.4 Insert `posture_change` notification into inbox when delta exceeds threshold (>2% or >0 findings)
- [x] 6.5 Insert `evidence_arrival` notification for each debounced batch (regardless of posture check)

## 7. Chat Context Enhancement

- [x] 7.1 Update `ChatAssistant` to read `selectedPolicyId` and active tab/view on open
- [x] 7.2 Include policy context in first message metadata when policy is active
- [x] 7.3 Skip context injection when no policy is selected (agent asks user)

## 8. Risk Severity Overlay (Phase 2)

- [x] 8.1 Create `GET /api/risks/severity?policy_id={id}` handler — join risks → risk_threats → control_threats → controls
- [x] 8.2 Add `RiskSeverityStore` interface with `GetPolicySeverity` method
- [x] 8.3 Implement SQL query for per-control max severity aggregation
- [x] 8.4 Add risk severity badge to `PostureCard` — show highest severity for failing controls
- [x] 8.5 Add risk indicator column to requirement matrix view
- [x] 8.6 Fetch severity data alongside posture/requirements data (parallel API calls)

## 9. Verification

- [x] 9.1 Sidebar shows exactly 4 items: Posture, Policies, Evidence, Inbox
- [x] 9.2 Clicking posture card navigates to `#/posture/{policy_id}` with breadcrumb and tabs
- [x] 9.3 `#/audit-history` redirects to `#/posture`
- [x] 9.4 Inbox badge shows unread count, decrements on item open
- [x] 9.5 `go vet ./...` passes, `helm template` renders clean
- [x] 9.6 `go test ./internal/...` passes with new store methods

---

## QE Instructions

### Happy Path

| Step | Action | Expected Result |
|:-----|:-------|:----------------|
| 1 | Open workbench, verify sidebar | Exactly 4 items: Posture, Policies, Evidence, Inbox |
| 2 | Navigate to Posture view | Cards show target count, control count, evidence freshness ("2h ago" style), and Owner field |
| 3 | Click a posture card "View Details" | Navigates to `#/posture/{policy_id}`, breadcrumb shows "Posture > [Policy Title]", three tabs visible: Requirements, Evidence, History |
| 4 | Switch between tabs | Each tab loads data filtered to the active policy; no standalone policy selector shown |
| 5 | Navigate to `#/audit-history` via URL bar | Redirects to `#/posture` |
| 6 | Navigate to `#/draft-review` via URL bar | Redirects to `#/inbox` |
| 7 | Click Inbox in sidebar | Inbox view loads; shows draft audit logs and notifications sorted by creation time |
| 8 | Click a draft in Inbox | Detail panel opens with YAML preview, reviewer edits (type override, notes), and Promote button |
| 9 | Open chat while viewing a policy drill-down | Chat context includes `policy_id` and `active_tab` in the dashboard context metadata |
| 10 | Verify risk severity badge on posture cards | Cards with failing controls linked to risks show a colored severity badge (Critical/High/Medium/Low) |
| 11 | Open Requirements tab, verify Risk column | Each requirement row shows a risk severity badge or "—" if no linked risks |
| 12 | Verify `helm template studio charts/complytime-studio/` | Renders NATS Deployment+Service when `nats.enabled=true`; internal gateway NP selects `studio-assistant` + `component: assistant` + `part-of: complytime-studio` |

### Edge Cases

| Case | Action | Expected Result |
|:-----|:-------|:----------------|
| No evidence for a policy | View posture card | Shows "No evidence yet" instead of freshness; target/control counts are 0 |
| No owner in policy YAML | View posture card | Shows "No owner" with muted styling |
| No risks imported | View posture cards and requirement matrix | No risk badges shown; Risk column shows "—" for all rows |
| NATS disabled (`nats.enabled: false`) | Start gateway | Warning: "nats connection failed — event-driven posture checks disabled"; inbox shows only draft audit logs, no event notifications |
| NATS enabled but no evidence events | Check inbox | No evidence_arrival or posture_change notifications; only drafts appear |
| Large policy set (10+ policies) | Load posture view | All cards render with inventory data; no timeout (enrichPostureOwners N+1 may be slow — monitor) |
| Draft already promoted | Click Promote again | 409 Conflict: "draft already promoted" |
| NP enforcement warning | Start gateway without `NETWORKPOLICY_ENFORCED` env | Warning logged: "NETWORKPOLICY_ENFORCED is unset — internal port has no auth" |
| Draft list pagination | Call `GET /api/draft-audit-logs?limit=5` | Returns at most 5 results; `limit=0` defaults to 100; `limit=5000` clamped to 1000 |
| Notification mark-read as non-admin | PATCH `/api/notifications/{id}/read` without admin role | Returns 403 (known limitation — viewer cannot mark read when auth + admin guard active) |

### Helm Verification

| Check | Command | Expected Result |
|:------|:--------|:----------------|
| CRD renders | `helm template studio charts/complytime-studio/` | Gateway has `public` (8080) and `internal` (8081) ports; `INTERNAL_PORT` env set |
| NATS gated | `helm template studio charts/complytime-studio/ --set nats.enabled=true` | `studio-nats` Deployment, Service, and NetworkPolicy rendered |
| NATS off | `helm template studio charts/complytime-studio/` | No NATS resources rendered (default `nats.enabled: false`) |
| NP tightened | Inspect `studio-allow-gateway-internal` | Source selector requires `name: studio-assistant`, `component: assistant`, `part-of: complytime-studio` |
| Assistant labels | Inspect `studio-assistant` Deployment | Pod template has `app.kubernetes.io/name: studio-assistant` label |
| Internal Service | Inspect rendered Services | `studio-gateway-internal` ClusterIP on port 8081 exists |
