# Posture Lifecycle

Continuous posture assessment: auto-trigger on evidence ingest, human override via AuditLog, unified compliance views.

## Capabilities

### 1. Auto-trigger Posture-Check on Evidence Ingest

**Trigger:** `POST /api/evidence` (REST) or OTel ingest path successfully inserts rows.

**Behavior:**

1. After evidence INSERT succeeds, collect distinct `policy_id` values from the batch.
2. For each policy_id, enqueue an async posture-check request.
3. Deduplicate: if a posture-check for the same policy_id was enqueued within the last 30 seconds, skip.
4. The posture-check calls the agent's A2A endpoint with a structured request: `"Run posture-check for policy {policy_id}, audit window: last 90 days"`.
5. The agent emits an `EvidenceAssessment` artifact. The existing interceptor persists it.

**Failure modes:**

| Condition | Behavior |
|:--|:--|
| Agent unavailable | Evidence stored. Posture-check skipped. `unassessed` count grows in `policy_posture`. |
| Agent returns error | Log warning. No assessment written. Retry on next evidence batch for same policy. |
| Policy not imported | Posture-check runs but agent reports "policy not found." Evidence remains, no assessment. Surfaced via orphan warning (Decision 10). |

**Cold start:** First deployment with existing evidence triggers a one-time sweep: `SELECT DISTINCT policy_id FROM evidence` → enqueue posture-check for each.

**Configuration:**

| Parameter | Default | Description |
|:--|:--|:--|
| `POSTURE_AUTO_TRIGGER` | `true` | Enable/disable auto-trigger |
| `POSTURE_DEDUP_WINDOW` | `30s` | Minimum interval between triggers for same policy |
| `POSTURE_AUDIT_WINDOW` | `90d` | Default lookback window for auto-triggered checks |

### 2. Human Override via AuditLog Finalization

**Trigger:** AuditLog artifact persisted by the Gateway interceptor (`tryPersistAuditLog`).

**Behavior:**

1. After inserting the AuditLog into `audit_logs`, parse `results[]` from the YAML content.
2. For each result with evidence references:
   a. If `assessment-override` field is present and is a valid 7-state classification, use it.
   b. If absent, map the result `type` to a classification:

   | `type` | Classification |
   |:--|:--|
   | Strength | Healthy |
   | Finding | Failing |
   | Gap | Blind |
   | Observation | *(skip — not a final judgment)* |

3. Resolve evidence rows: match result's `evidence[].collected` + `criteria-reference` back to `evidence.evidence_id` via ClickHouse query.
4. INSERT into `evidence_assessments`:
   - `assessed_by`: `audit:<audit_id>`
   - `source_audit_id`: the AuditLog's `audit_id`
   - `classification`: from step 2
   - `reason`: the result's `description`

**Schema change — `evidence_assessments`:**

```sql
ALTER TABLE evidence_assessments
    ADD COLUMN IF NOT EXISTS source_audit_id Nullable(String)
```

**Agent behavior change — posture-check skill:**

When classifying evidence, check if the latest assessment (via `FINAL`) has `assessed_by LIKE 'audit:%'`. If so, preserve the human override — do not re-classify. Emit the existing classification in the EvidenceAssessment artifact with a note: `"Human override from audit {source_audit_id}"`.

### 3. Unified Compliance Views

**`unified_compliance_state`** — row-level, used for drill-down:

```sql
CREATE VIEW unified_compliance_state AS
SELECT
    e.evidence_id, e.policy_id, e.control_id, e.target_id,
    e.target_name, e.eval_result, e.collected_at, e.attestation_ref,
    a.classification, a.reason, a.assessed_by, a.assessed_at,
    a.source_audit_id
FROM evidence e
LEFT JOIN (SELECT * FROM evidence_assessments FINAL) a
    ON e.evidence_id = a.evidence_id
```

**`policy_posture`** — aggregated, used for summaries:

```sql
CREATE VIEW policy_posture AS
SELECT
    e.policy_id, p.title AS policy_title, e.target_id,
    countIf(a.classification = 'Healthy') AS healthy,
    countIf(a.classification = 'Failing') AS failing,
    countIf(a.classification = 'Wrong Source') AS wrong_source,
    countIf(a.classification = 'Wrong Method') AS wrong_method,
    countIf(a.classification = 'Unfit Evidence') AS unfit,
    countIf(a.classification = 'Stale') AS stale,
    countIf(a.classification = 'Blind') AS blind,
    countIf(a.classification IS NULL) AS unassessed,
    max(e.collected_at) AS latest_evidence,
    max(a.assessed_at) AS latest_assessment
FROM evidence e
LEFT JOIN (SELECT * FROM evidence_assessments FINAL) a
    ON e.evidence_id = a.evidence_id
LEFT JOIN policies p ON e.policy_id = p.policy_id
GROUP BY e.policy_id, p.title, e.target_id
```

**Consumers:**

| Consumer | View | Use |
|:--|:--|:--|
| Agent (quick posture summary) | `policy_posture` | "How's our posture?" → instant answer without full posture-check |
| Agent (drill-down) | `unified_compliance_state` | "Why is control X failing?" → row-level detail with assessment context |
| Future REST API | `policy_posture` | `GET /api/posture` → JSON for workbench dashboard |
| Staleness detection | `policy_posture` | `WHERE latest_assessment < now() - INTERVAL 7 DAY` |

## Data Flow

```
complyctl scan
     │
     ▼
POST /api/evidence ──────────────────────── INSERT evidence (sync)
     │                                            │
     │                                      ┌─────┴─────┐
     │                                      │ policy_id  │
     │                                      │ dedup 30s  │
     │                                      └─────┬─────┘
     │                                            │ async
     │                                            ▼
     │                                   Agent posture-check
     │                                            │
     │                                            ▼
     │                                   EvidenceAssessment artifact
     │                                            │
     │                                            ▼
     │                                   INSERT evidence_assessments
     │                                   (assessed_by = agent)
     │
     ▼
policy_posture VIEW ◄── query-time JOIN ──── evidence + evidence_assessments
     │
     ▼
Agent or REST consumer reads posture summary
     │
     ▼
Auditor requests AuditLog ──────────────── Agent creates AuditLog
                                                  │
                                                  ▼
                                           INSERT audit_logs
                                                  │
                                                  ▼
                                           Extract results[] overrides
                                                  │
                                                  ▼
                                           INSERT evidence_assessments
                                           (assessed_by = audit:<id>)
                                                  │
                                                  ▼
                                           policy_posture reflects
                                           human-confirmed state
```

### 4. Posture Worker Service

Separate stateless Go service that receives trigger notifications from the Gateway and drives posture-check assessments via A2A. Gateway stays a thin proxy — it fires one HTTP POST and moves on.

**Notification path:**

```
Gateway (after evidence INSERT):
  POST http://studio-posture-worker:8081/trigger
  Body: {"policy_ids": ["ampel-branch-protection"], "source": "evidence-ingest"}
  → fire-and-forget, log warning on failure

Worker:
  → dedup: skip if policy_id triggered < POSTURE_DEDUP_WINDOW ago
  → enqueue to internal work queue
  → return 202 Accepted
```

**Work queue processing:**

For each dequeued policy_id:
1. POST A2A `tasks/send` to agent with message: `"Run posture-check for policy {policy_id}, audit window: last {POSTURE_AUDIT_WINDOW}"`
2. Consume SSE response stream
3. Parse `EvidenceAssessment` artifact from stream
4. Validate classifications against `ValidClassifications` enum
5. INSERT into `evidence_assessments` via shared persistence logic

**Shared assessment package:**

Extract interceptor logic into `internal/assessment/`:

```
internal/assessment/
├── parse.go      ← YAML parsing, classification validation
└── persist.go    ← InsertEvidenceAssessments, shared by Gateway + Worker

internal/agents/
└── artifact_interceptor.go  ← imports assessment.Parse + assessment.Persist

cmd/posture-worker/
└── main.go                  ← imports assessment.Parse + assessment.Persist
```

**Lifecycle:**

| Phase | Behavior |
|:--|:--|
| Startup | Connect ClickHouse. Cold-start sweep: `SELECT DISTINCT policy_id FROM evidence e LEFT JOIN evidence_assessments a ON e.evidence_id = a.evidence_id WHERE a.evidence_id IS NULL` → enqueue each. Start HTTP on :8081. |
| Running | Listen for `/trigger` POSTs. Process work queue with configurable concurrency. Dedup in-memory by policy_id + timestamp. |
| Shutdown | Drain in-flight work. Graceful HTTP shutdown. |

**Configuration:**

| Parameter | Default | Description |
|:--|:--|:--|
| `AGENT_A2A_URL` | `http://studio-assistant:8080` | Direct agent A2A endpoint |
| `KAGENT_A2A_URL` | *(empty)* | kagent controller URL (overrides direct) |
| `KAGENT_AGENT_NAMESPACE` | `default` | Namespace for controller path |
| `CLICKHOUSE_ADDR` | *(required)* | ClickHouse connection |
| `CLICKHOUSE_USER` | `default` | ClickHouse auth |
| `CLICKHOUSE_PASSWORD` | *(required)* | ClickHouse auth |
| `POSTURE_DEDUP_WINDOW` | `30s` | Min interval between triggers per policy |
| `POSTURE_AUDIT_WINDOW` | `90d` | Default lookback for auto-triggered checks |
| `POSTURE_CONCURRENCY` | `2` | Max parallel posture-checks |

**Gateway integration:**

Gateway gets one new env var: `POSTURE_WORKER_URL`. If unset, auto-trigger is disabled — evidence ingest works normally, assessments don't auto-refresh.

```
Gateway (evidence handler):
  if POSTURE_WORKER_URL != "" {
    POST {POSTURE_WORKER_URL}/trigger with distinct policy_ids from batch
  }
```

**Helm deployment:**

New template: `posture-worker.yaml` (Deployment + ClusterIP Service on port 8081).

**DNS:** All services in the same namespace. Worker resolves `studio-assistant:8080` and `studio-clickhouse:9000` via standard ClusterIP DNS. Gateway resolves `studio-posture-worker:8081`. No cross-namespace resolution.

## Dependencies

| Dependency | Status |
|:--|:--|
| `evidence_assessments` table | Exists (migration 5) |
| AuditLog interceptor | Exists (`artifact_interceptor.go`) |
| EvidenceAssessment interceptor | Exists (`artifact_interceptor.go`) |
| Posture-check skill | Exists (`skills/posture-check/SKILL.md`) |
| `source_audit_id` column | New (migration 6) |
| `unified_compliance_state` VIEW | New (migration 7) |
| `policy_posture` VIEW | New (migration 8) |
| `internal/assessment/` shared package | New — extracted from `artifact_interceptor.go` |
| `cmd/posture-worker/` service | New |
| `posture-worker.yaml` Helm template | New |
| `POSTURE_WORKER_URL` Gateway env var | New |
| Gateway trigger POST in evidence handler | New |
