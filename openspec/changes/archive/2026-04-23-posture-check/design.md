## Context

The assistant agent's only workflow path is audit production — load Policy, query evidence, classify, produce AuditLog. Users discover evidence quality problems (wrong scanner, stale data, missing cycles) only when they request an audit. The Policy schema already defines assessment plans with frequency, executor, and mode. The evidence table already stores `engine_name`, `engine_version`, and `plan_id`. The join between them is not implemented anywhere.

## Goals / Non-Goals

**Goals:**
- Agent answers "what's my posture for policy X?" without producing an AuditLog
- Agent extracts assessment plans from Policy YAML and validates each against the evidence stream
- Agent checks executor provenance (`engine_name` vs `evaluation-methods[].executor.id`)
- Agent detects cadence gaps using the same frequency logic the audit skill already defines
- Agent returns a per-plan readiness table with actionable classification

**Non-Goals:**
- Autonomous background monitoring (Phase 2 — separate change)
- New ClickHouse tables or schema migrations
- New MCP tools
- Evidence intake gating at ingest time (depends on Phase 2 event-driven architecture)
- Modifying EvaluationLog or AuditLog artifact structure

## Decisions

### D1: Separate skill, not inlined in audit skill

**Choice:** Create `skills/posture-check/SKILL.md` as a standalone skill rather than expanding `skills/studio-audit/SKILL.md`.

**Why:** The audit skill is already dense (classification, cadence, coverage mapping). Posture check is a different concern — plan-level readiness vs. control-level evidence synthesis. Separate skills keep each focused and independently referenceable. Both share the same ClickHouse tables and cadence logic but serve different workflow paths.

**Alternative considered:** Merge into studio-audit. Rejected — makes the audit skill responsible for two distinct outputs (AuditLog authoring vs. readiness table) and bloats context window for both paths.

### D2: Prompt-level routing, not tool-based dispatch

**Choice:** Add a routing step to the prompt that detects posture-check intent (keywords: "posture", "readiness", "status", "how ready", "assessment plan") and dispatches to the posture workflow. No new ADK tool.

**Why:** The posture check workflow uses the same tools the agent already has (`run_select_query`, Policy YAML parsing). The difference is workflow shape, not tooling. Routing at the prompt level avoids adding unnecessary tool surface area.

**Alternative considered:** New `check_posture` ADK tool that runs the join server-side. Rejected for Phase 1 — adds implementation complexity when the agent can compose existing tools. Revisit if query composition proves unreliable.

### D3: Five-state classification per assessment plan

**Choice:** Each assessment plan is classified into one of five states:

| State | Condition |
|:--|:--|
| Healthy | Evidence exists, on cadence, correct executor, latest result Passed |
| Failing | Evidence exists, correct executor, latest result Failed or Needs Review |
| Wrong Source | Evidence exists but `engine_name` does not match plan's `executor.id` |
| Stale | Evidence exists from correct executor but outside the current frequency window |
| Blind | No evidence rows match the plan's `requirement_id` within the audit window |

**Why:** Maps directly to the auditor's mental model from our exploration. "Wrong Source" is the provenance check — evidence exists but is inadmissible. "Stale" vs "Blind" distinguishes "we had it but it expired" from "we never collected it." Each state implies a different remediation action.

**Priority order when multiple conditions apply:** Blind > Wrong Source > Stale > Failing > Healthy. Worst state wins.

### D4: Reuse cadence logic from studio-audit skill

**Choice:** The posture-check skill references the same frequency-to-cycle-length mapping defined in `skills/studio-audit/SKILL.md` (daily=1d, weekly=7d, monthly=30d, quarterly=90d, annually=365d).

**Why:** Single source of truth for cadence interpretation. If the audit skill's cadence logic changes, posture checks stay consistent.

### D5: Policy YAML parsing in-prompt, not pre-extracted

**Choice:** The agent parses `policies.content` YAML at query time to extract `adherence.assessment-plans[]`. No pre-parsed `assessment_plans` table.

**Why:** The `policies` table stores the full Policy YAML in `content`. Adding a parsed table would require schema changes and an ingestion pipeline update — out of scope. The agent already parses Policy YAML for catalog imports in the audit workflow. Same pattern, different extraction target.

## Risks / Trade-offs

**[Risk] LLM misparses nested Policy YAML** → The `adherence.assessment-plans` structure is 3-4 levels deep. Mitigated by the skill providing an explicit extraction template showing the exact path and expected fields. If unreliable in practice, escalate to D2 alternative (server-side tool).

**[Risk] `engine_name` not populated in evidence rows** → Some ingestion paths may leave `engine_name` NULL. The skill instructs the agent to classify NULL engine as "Unknown Source" (distinct from Wrong Source) and note it in the readiness table.

**[Risk] `plan_id` not populated in evidence rows** → If evidence was ingested before assessment plans were defined, `plan_id` may be NULL. The skill instructs fallback to matching by `requirement_id` alone when `plan_id` is absent.

**[Trade-off] No real-time gating** → Phase 1 is on-demand only. Bad evidence can still accumulate between posture checks. Acceptable — the check is fast enough to run frequently, and Phase 2 will add autonomous monitoring.
