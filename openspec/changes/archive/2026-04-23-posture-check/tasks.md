## 1. Posture-Check Skill

- [x] 1.1 Create `skills/posture-check/SKILL.md` with frontmatter (`name: posture-check`, `description`)
- [x] 1.2 Add assessment-plan extraction section: document the YAML path (`adherence.assessment-plans[]`) and fields to extract (`id`, `requirement-id`, `frequency`, `evaluation-methods[].executor`, `mode`, `evidence-requirements`)
- [x] 1.3 Add evidence query template: ClickHouse query pattern joining `requirement_id`, `plan_id`, `policy_id`, `target_id` with frequency-derived time window
- [x] 1.4 Add provenance validation section: `engine_name` vs `executor.id` comparison logic, NULL handling as "Unknown Source"
- [x] 1.5 Add five-state classification table: Healthy, Failing, Wrong Source, Stale, Blind — with conditions and priority order
- [x] 1.6 Add cadence reference: point to studio-audit frequency mapping (daily=1d, weekly=7d, monthly=30d, quarterly=90d, annually=365d)

## 2. Prompt Update

- [x] 2.1 Add routing step to `agents/assistant/prompt.md`: detect posture-check intent vs. audit production, with disambiguation prompt for ambiguous requests
- [x] 2.2 Add posture-check workflow section to prompt: Load Policy → Extract assessment plans → Discover targets → Query evidence per plan per target → Classify → Return readiness table
- [x] 2.3 Add readiness table output format: Plan ID, Frequency, Last Evidence, Source Match, Latest Result, Classification columns with summary line

## 3. Agent Registration

- [x] 3.1 Add `- path: skills/posture-check` to `agents/assistant/agent.yaml` skills array

## 4. Verification

- [x] 4.1 Run `make sync-skills` and confirm posture-check appears in `agents/assistant/skills/` (BYO agent — no kagent gitRefs)
- [~] 4.2 Test via workbench: ask "What's my posture for policy X?" and verify the agent returns a readiness table (not an AuditLog) — *Skipped: manual verification, not automatable in CI*
- [~] 4.3 Test disambiguation: ask an ambiguous question and verify the agent asks for clarification — *Skipped: manual verification, not automatable in CI*
