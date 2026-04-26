## Why

The assistant agent produces AuditLogs reactively — only when an auditor requests one. By that point, evidence quality problems (wrong scanner, stale collections, missing assessment cycles) surface under audit pressure, forcing a scramble to re-collect. The Policy already defines assessment plans with frequency, executor, and mode requirements. The evidence table already captures `engine_name` and `engine_version`. Nothing joins them to give a pre-audit readiness signal.

## What Changes

- **New assistant workflow path**: "posture check" — user asks for compliance posture against a policy, agent returns a readiness table per assessment plan without producing an AuditLog.
- **New skill**: teaches the agent to extract `adherence.assessment-plans[]` from Policy YAML, query evidence by requirement/plan/target, and compare `engine_name`/`engine_version` against the plan's `evaluation-methods[].executor`.
- **Prompt update**: add a routing step that distinguishes "posture check" requests from "audit production" requests, dispatching to the appropriate workflow.
- **Classification expansion**: each assessment plan is classified as Healthy, Failing, Wrong Source, Stale, or Blind based on evidence presence, cadence, provenance, and result.

## Capabilities

### New Capabilities
- `posture-check-skill`: Skill that teaches the agent to extract assessment plans from Policy YAML, join against evidence rows, validate executor provenance, check cadence against frequency, and classify plan-level readiness.
- `posture-check-workflow`: Prompt workflow path that routes posture-check requests, invokes the skill, and returns a per-plan readiness table with actionable flags.

### Modified Capabilities
- `agent-spec-skills`: Agent skill list updated to include the new posture-check skill.

## Impact

- `skills/posture-check/SKILL.md` — new skill file
- `agents/assistant/prompt.md` — add posture-check workflow path and routing logic
- `agents/assistant/agent.yaml` — register new skill reference
- No schema changes — `engine_name`/`engine_version` and `plan_id` already exist in the evidence table
- No new MCP tools — `run_select_query` is sufficient
