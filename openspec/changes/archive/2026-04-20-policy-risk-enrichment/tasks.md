## 1. Risk Reasoning Skill

- [x] 1.1 Create `skills/risk-reasoning/SKILL.md` with domain knowledge: appetite vs tolerance semantics, prioritization signals (threat density, vector breadth, capability exposure, tolerance cap checks), catalog-vs-policy data boundary, residual risk pattern
- [x] 1.2 Include severity justification criteria with example signal-to-severity mappings
- [x] 1.3 Include the residual risk flow pattern (inherent catalog → policy → residual catalog → policy update)

## 2. Policy-Composer Agent Updates

- [x] 2.1 Add `risk-reasoning` skill reference to `agents/policy-composer/agent.yaml`
- [x] 2.2 Update `agents/policy-composer/prompt.md` to add enrichment opt-in prompt before Phase 1
- [x] 2.3 Update Phase 1 Step 2 (Derive Risk Entries) with enriched path: threat graph traversal, severity justification via signals, impact narrative generation
- [x] 2.4 Add prioritization summary table step between Step 2 and Step 3 (enriched path only)
- [x] 2.5 Add tolerance cap violation flagging in prioritization summary
- [x] 2.6 Add optional residual risk identification step after Phase 2 Step 7 (risk-to-control linkage)

## 3. Sync and Validate

- [x] 3.1 Run `make sync-prompts` to copy updated prompt into Helm chart
- [x] 3.2 Validate that declining enrichment produces identical behavior to current fast path
- [x] 3.3 Sync specs to `openspec/specs/risk-reasoning/spec.md` and `openspec/specs/job-lifecycle/spec.md`
