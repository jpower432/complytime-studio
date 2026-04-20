## Context

The policy-composer agent runs a two-phase conversation: Phase 1 produces a RiskCatalog, Phase 2 produces a Policy. Phase 1 currently derives risks from threats mechanically (one risk per threat, severity assigned without structured reasoning). The Gemara schema captures qualitative signals (threat density, vector breadth, appetite, tolerance) that are not surfaced today.

The user's domain knowledge (provided as input to this proposal) defines a clear data boundary: catalogs describe risk scenarios context-free; policies bind them to specific inventories. Risk reasoning bridges these layers by producing justified severity assignments and identifying tolerance violations before policy treatment decisions.

The policy-composer already references two skills: `gemara-layers` (schema knowledge) and `policy-risk-linkage` (join logic). Adding a `risk-reasoning` skill follows the established pattern.

## Goals / Non-Goals

**Goals:**
- Enrich the policy-composer's Phase 1 with structured risk reasoning when the user opts in
- Produce RiskCatalogs where every severity has a traceable justification chain (risk → threats → vectors → capabilities)
- Surface tolerance cap violations as the primary prioritization signal
- Enable residual risk identification as an optional sub-step after control mapping
- Keep the existing fast-path Phase 1 intact for users who decline enrichment

**Non-Goals:**
- Numeric risk scoring (Gemara deliberately excludes computed scores from catalogs)
- Deterministic tooling in `gemara-mcp` for graph traversal (future consideration, not blocking)
- Changes to the RiskCatalog or Policy Gemara schema
- A new agent or A2A skill registration
- Automated residual risk catalog generation without user input

## Decisions

| # | Decision | Rationale | Alternatives Considered |
|:--|:---------|:----------|:------------------------|
| 1 | Implement as a skill (`skills/risk-reasoning/SKILL.md`), not a prompt change | Domain knowledge (appetite semantics, prioritization signals, residual risk pattern) is reusable. Other agents (e.g., gap-analyst) could reference the same knowledge. | (a) Inline everything in `prompt.md` — couples domain knowledge to one agent's workflow. |
| 2 | Opt-in via conversation, not configuration | The agent asks "Do you want risk analysis included?" at the start of Phase 1. This keeps the interaction natural and avoids Helm/CRD changes. | (a) Helm value `riskEnrichment.enabled` — too rigid, mixes deployment config with workflow preference. (b) Always-on — increases token cost and conversation length for users who don't need it. |
| 3 | Enrichment modifies Phase 1 steps, not the phase structure | Steps 1-3 remain (derive categories → derive entries → validate). Enrichment adds depth to Step 2: threat graph traversal, severity justification, tolerance checks. No new steps. | (a) Add a Phase 0 before Phase 1 — increases conversation length and breaks the existing step numbering. |
| 4 | Severity justification uses threat graph signals, not LLM judgment alone | The skill defines countable signals (threat density, vector breadth, capability exposure) that the LLM evaluates from the ThreatCatalog structure. This grounds severity in data rather than opinion. | (a) LLM assigns severity with free-text reasoning — no structure, hard to audit. (b) Deterministic scoring via MCP tool — ideal long-term, but requires `gemara-mcp` changes. |
| 5 | Residual risk is a separate conversation turn, not automatic | After Phase 2's risk-to-control linkage, the agent presents which risks remain partially or fully unmitigated. The user decides whether to author a residual risk catalog. | (a) Always generate residual catalog — presumes the user wants it. (b) Skip entirely — misses a key use case. |
| 6 | Tolerance cap check is presented as a table, not enforced | The skill instructs the agent to compare `risk.severity` against `group.max-severity` and flag violations. The agent presents violations; the user decides treatment. | (a) Hard enforcement (reject risks above tolerance) — defeats the purpose of documenting risks that exceed tolerance for escalation. |

## Risks / Trade-offs

| Risk | Severity | Mitigation |
|:-----|:---------|:-----------|
| Enriched path increases conversation length by 2-3 exchanges | Low | Opt-in only. Agent batches graph analysis into one summary table. |
| LLM may produce inconsistent severity reasoning across sessions | Medium | Skill provides explicit decision criteria (threat density thresholds, severity-to-signal mapping). Validation via `validate_gemara_artifact` catches structural errors. |
| Users may not understand appetite vs tolerance distinction | Low | Skill includes definitions and examples. Agent presents defaults with explanations. |
| Residual risk sub-step may confuse the conversation flow | Low | Agent explicitly offers it as optional after Phase 2. Clear transition prompt. |

## Implementation Notes

All 12 tasks implemented.

| Area | What was delivered |
|---|---|
| `skills/risk-reasoning/SKILL.md` | Domain knowledge: appetite vs tolerance (ISO 31000), 5 prioritization signals, severity heuristics (Critical/High/Medium/Low with signal thresholds), catalog-vs-policy boundary, 5-step residual risk flow |
| `agents/policy-composer/agent.yaml` | Added `- path: skills/risk-reasoning` to skills array |
| `agents/policy-composer/prompt.md` | Opt-in prompt before Phase 1; enriched Step 2 (threat graph traversal, severity justification, impact narratives); Step 2b (prioritization summary table with tolerance violations); Step 7a (optional residual risk catalog after linkage, independent of enrichment choice) |
| `agent-specialists.yaml` | Added `gitRefs` entry for `skills/risk-reasoning` on `studio-policy-composer` |
| `openspec/specs/` | `risk-reasoning/spec.md` synced; `job-lifecycle/spec.md` merged with enrichment scenario |

**Design evolution**: Step 7a (residual risk identification) was made independent of the enrichment opt-in -- identifying unmitigated risks is valuable regardless of whether severity was deeply justified.
