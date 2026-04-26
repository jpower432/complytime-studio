## Context

Studio serves an audit liaison — someone who prepares evidence, works with system owners, and presents to auditors. The current UX is pull-based: the user searches for data, triggers audits manually, and navigates disconnected views. Evidence arrives from pipelines (OPA, CSPM tools, OTel) but nothing reacts to it. The sidebar has six items with unclear relationships. "Review" doesn't convey its purpose, "Audit History" is a per-policy concern stranded at the top level, and posture cards lack inventory context.

The gateway is a Go modulith. The frontend is a Preact SPA embedded via `go:embed`. Evidence arrives via `cmd/ingest` (OTel HTTP receiver) and `POST /api/evidence`. ClickHouse stores everything. The assistant agent communicates via A2A proxy and uses MCP tools for ClickHouse queries and Gemara validation.

## Goals / Non-Goals

**Goals:**
- Restructure navigation around the audit liaison's mental model: inventory → drill-down → act
- Make evidence arrival observable — push-based reactions via NATS event bus
- Surface agent work in a unified inbox with badge count
- Add inventory context (targets, controls, owners, freshness) to posture cards
- Enable risk severity overlay on posture and requirements (Phase 2)

**Non-Goals:**
- Full RBAC / multi-tenancy (separate decision: authorization-model.md)
- Real-time WebSocket push to the browser (polling + SSE is sufficient for now)
- Replacing the chat FAB with an embedded panel (chat stays as overlay)
- Building a custom NATS operator or cluster (use single-node NATS in-chart)

## Decisions

### 1. NATS for evidence event bus

**Choice:** NATS (single-node, in-chart) over ClickHouse polling or Kafka.

| Alternative | Why Not |
|:--|:--|
| ClickHouse polling (materialized view + gateway goroutine) | Adds query load to CH on every poll cycle. Latency floor = poll interval. Works but doesn't scale. |
| Kafka | Overkill for event volume. Heavier ops burden (ZooKeeper/KRaft). Not justified until multi-cluster. |
| Redis Pub/Sub | Another stateful dependency. NATS is lighter and CNCF-graduated. |
| CloudEvents over HTTP | No persistence. Requires the gateway to be up when ingest fires. |

NATS is CNCF-graduated, Go-native (`nats.go`), and runs as a single binary. In-chart deployment: one `Deployment` + `Service`, ~20MB memory. JetStream disabled initially — pure pub/sub is sufficient.

**Subject scheme:** `studio.evidence.{policy_id}` — gateway subscribes to `studio.evidence.*`.

### 2. Inbox as unified notification surface

**Choice:** Single "Inbox" view replaces "Review" (draft-review-view). Inbox shows three types of items:

| Type | Source | Example |
|:--|:--|:--|
| Draft audit log | Agent `publish_audit_log` tool | "Agent produced an audit log for ampel-branch-protection" |
| Posture change | Agent posture-check triggered by evidence event | "Pass rate for policy X dropped to 72% (was 89%)" |
| Evidence arrival | NATS event, summarized by gateway | "14 new evidence records for ampel-branch-protection in the last hour" |

All items stored in a `notifications` ClickHouse table with `type`, `policy_id`, `payload JSON`, `read Boolean`, `created_at`. Badge count = unread notifications.

### 3. Posture drill-down replaces standalone Audit History

**Choice:** Breadcrumb pattern. Clicking a posture card opens a policy detail view with three tabs: Requirements, Evidence Timeline, Audit History. Sidebar item "Audit History" removed.

```
Posture → [ampel-branch-protection] → Requirements | Evidence | History
```

URL hash: `#/posture/ampel-branch-protection?tab=history`

The `AuditHistoryView` component moves from a top-level route to a tab within `PolicyDetailView`. No backend changes — same API endpoints, different routing.

### 4. Inventory cards with RACI context

**Choice:** Enhance posture cards with data already in ClickHouse or derivable from the Policy artifact:

| Field | Source | Display |
|:--|:--|:--|
| Target count | `COUNT(DISTINCT target_id) FROM evidence WHERE policy_id = ?` | "12 targets" |
| Control count | `COUNT(*) FROM controls WHERE policy_id = ?` | "47 controls" |
| Last evidence | `max(collected_at) FROM evidence WHERE policy_id = ?` | "2h ago" |
| Owner | `Policy.contacts` RACI (Accountable) — parsed at import time if `policy_contacts` table exists (authorization-model Phase 1), else from YAML | "platform-team" |

Single new API endpoint: `GET /api/posture/summary` returns enriched posture rows with these fields. Replaces current `GET /api/posture` or extends it.

### 5. Evidence event → agent trigger flow

```
cmd/ingest                    NATS                   Gateway                    Agent
    │                          │                       │                          │
    ├─ insert evidence ───────►│                       │                          │
    ├─ publish studio.evidence.{pid} ►│                │                          │
    │                          ├──────► subscribe ─────┤                          │
    │                          │       debounce 30s    │                          │
    │                          │       ├─ POST /a2a ──►│──► posture-check skill   │
    │                          │       │               │    (lightweight, no full │
    │                          │       │               │     audit)               │
    │                          │       │               │◄── result ◄──────────────┤
    │                          │       │               ├─ insert notification ────┤
    │                          │       │               ├─ update posture cache    │
```

Debounce is critical — evidence arrives in batches. Gateway accumulates events per `policy_id` for 30s, then triggers one posture check per policy.

### 6. Risk severity overlay (Phase 2)

Risk data exists in `risks` and `risk_threats` tables. Overlay renders as:

- **Posture card:** small severity badge (Critical/High/Medium/Low) derived from highest-severity risk linked to failing controls
- **Requirement matrix row:** risk indicator column showing aggregate risk level per requirement

Query pattern: join `risks` → `risk_threats` → `threats` → `control_threats` → `controls` → requirement evidence. Materialized view or on-demand query TBD based on performance.

## Risks / Trade-offs

| Risk | Mitigation |
|:--|:--|
| NATS adds infra complexity | Single-node, in-chart, pure pub/sub. No JetStream. If NATS is down, evidence still ingests — notifications just delay until reconnect. |
| Agent posture-check storms on bulk import | 30s debounce per policy. Max 1 concurrent check per policy (gateway tracks in-flight). |
| Inbox table grows unbounded | TTL on `notifications` table (90 days). Read notifications auto-expire after 30 days. |
| Breadcrumb pattern breaks deep links | Hash scheme `#/posture/{policy_id}?tab=X` preserves deep-linkability. Existing `#/audit-history` redirects to `#/posture?tab=history` for backward compat. |
| RACI owner field empty for policies without contacts | Show "No owner" with a hint to add contacts to the Policy artifact. Non-blocking. |

## Open Questions

- Should evidence arrival notifications aggregate per policy or per control? Per-policy is simpler but less actionable.
- Should the agent posture-check result replace the posture card data or coexist as a separate "agent assessment" view?
- When NATS is unavailable, should ingest block or fire-and-forget? Fire-and-forget preserves ingest reliability but loses the event.
- Should the inbox show browser notifications (Notification API) for critical posture changes, or is in-app badge sufficient?
