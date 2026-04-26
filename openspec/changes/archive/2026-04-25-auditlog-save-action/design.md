## Context

Real audit workflows are RFI-driven: auditor receives request, team gathers evidence, auditor reviews and classifies, auditor writes the report. The auditor owns the assertion. The agent should mirror this — evidence assembly and draft classification are the agent's job, review and promotion are the human's.

The previous design had the agent POST directly to `audit_logs` with a service account token. This created three problems:
1. No human review before persistence to the compliance record of truth
2. Agent needed admin-equivalent credentials (DB grooming vector)
3. Violated the "user controls what gets persisted" non-goal

## Goals / Non-Goals

**Goals:**
- Agent drafts AuditLogs with per-result reasoning visible to the human
- Agent emits a factual evidence package (no judgment) as a separate artifact
- Human reviews, overrides, and promotes drafts to official records
- Human's identity is on the official AuditLog, agent credited as tool

**Non-Goals:**
- Auto-saving without human action
- Agent writing to `audit_logs` directly
- Agent having admin or service-account credentials to the Gateway
- Full adversarial review (second agent) — deferred

## Decisions

### D1: Two-artifact workflow

**Choice:** The agent emits two artifacts per target: (1) Evidence Package (factual), (2) Draft AuditLog (judgment + reasoning).

**Why:** Separates fact from opinion. The evidence package is ground truth the human can verify independently. The draft is the agent's interpretation, subject to override.

### D2: Draft table, not direct persist

**Choice:** Agent writes to `draft_audit_logs`. Human promotes to `audit_logs` via `POST /api/audit-logs/promote`.

**Why:** The official audit record must carry human approval. Drafts are ephemeral workbench state. Promotion is an explicit human action with their identity attached.

### D3: Per-result reasoning

**Choice:** Each result in the draft AuditLog carries an `agent-reasoning` field explaining the classification.

**Why:** The human needs to evaluate the agent's judgment, not just the conclusion. "Strength because all 3 evals passed within cadence" is reviewable. "Strength" alone is not.

```yaml
results:
  - id: bp-3-01-complyctl
    title: Branch protection enforced
    type: Strength
    agent-reasoning: >-
      3 passing evaluations from engine complytime-scanner within the
      assessment window. Source matches policy executor BP-3.01.
      Latest evidence collected 2026-04-16.
    evidence:
      - type: EvaluationLog
        collected: "2026-04-16T10:00:00Z"
        description: Branch protection check via complytime-scanner
```

### D4: No agent credentials to Gateway write endpoints

**Choice:** The `publish_audit_log` tool writes to `/internal/draft-audit-logs` — an internal-only endpoint with no auth. Access restricted by Kubernetes NetworkPolicy to `studio-assistant` pods.

**Why:** The agent never receives tokens that could be misused. Network-level isolation replaces credential-based trust. The internal endpoint only accepts drafts, never official records.

### D5: Revert synthetic admin session

**Choice:** Remove the `service@complytime.local` synthetic session injected for API token requests. API tokens should not grant admin role by default.

**Why:** The API token admin bypass was a shortcut to get the direct-persist path working. With the draft pattern, the agent doesn't need write access to admin-protected endpoints.

### D6: Promote carries human identity

**Choice:** `POST /api/audit-logs/promote` requires an authenticated session. The promoting user's identity is recorded as `created_by` on the official AuditLog.

**Why:** Accountability. The human who approved the audit owns the assertion. The agent is credited in `model` and `prompt_version` fields.

## Data Flow

```
  Agent                        Gateway                    ClickHouse
    │                             │                           │
    │  POST /internal/            │                           │
    │  draft-audit-logs           │                           │
    │  {content, reasoning}       │                           │
    │────────────────────────────▶│  INSERT                   │
    │                             │  draft_audit_logs         │
    │                             │──────────────────────────▶│
    │                             │                           │
    │  save_artifact()            │                           │
    │  (ADK in-memory)            │                           │
    │                             │                           │

  Human (workbench)            Gateway                    ClickHouse
    │                             │                           │
    │  Reviews draft in UI        │                           │
    │  Overrides BP-4.01          │                           │
    │  Adds note to BP-1.01      │                           │
    │                             │                           │
    │  POST /api/audit-logs/      │                           │
    │  promote                    │                           │
    │  {draft_id, overrides}      │                           │
    │────────────────────────────▶│  INSERT                   │
    │                             │  audit_logs               │
    │  (session: user@co.com)     │  created_by: user@co.com  │
    │                             │──────────────────────────▶│
```

## Risks / Trade-offs

**[Risk] Agent skips evidence package, goes straight to draft** — Mitigated by prompt instructions. The evidence package step is explicit in the workflow.

**[Risk] Human rubber-stamps without reviewing** — Outside system control. Per-result reasoning lowers the barrier to review. Audit trail shows time-on-page for compliance programs that track reviewer engagement.

**[Risk] Draft table accumulates stale drafts** — Add TTL or periodic cleanup. Drafts older than 30 days auto-expire.

**[Risk] Internal endpoint reachable from compromised pod** — NetworkPolicy restricts ingress to `studio-assistant` only. Defense in depth: the endpoint only accepts drafts, never official records.
