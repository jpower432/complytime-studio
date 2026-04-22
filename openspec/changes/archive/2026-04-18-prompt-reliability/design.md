## Context

The assistant agent uses a single `LlmAgent` (Google ADK) with a static instruction string assembled at startup. `load_prompt()` reads `prompt.md`, then globs all `skills/*/SKILL.md` files and appends them with `---` separators. All skills are loaded on every turn.

The `audit_logs` table stores validated AuditLog artifacts with `created_by` (user identity) but no information about which model or prompt version produced the artifact.

The `audit-methodology` skill defines classification rules as prose tables (Strength/Finding/Gap/Observation, Satisfied/Partially/Not Satisfied). The `coverage-mapping` skill defines coverage derivation rules. Neither includes concrete examples showing the classification applied to real evidence patterns.

## Goals / Non-Goals

**Goals:**
- Few-shot examples for all three classification decisions (result type, satisfaction, coverage status)
- AuditLog provenance captured at storage time (model, prompt version)

**Non-Goals:**
- Changing `load_prompt()` merge semantics or prompt assembly structure (keep existing concatenation)
- Selective/dynamic skill loading (all skills load every turn, same as today)
- Multi-model consensus or self-consistency sampling (evaluate after few-shot baseline)
- Changing classification definitions or thresholds (skill content unchanged)
- Modifying the Gemara `#AuditLog` CUE schema

## Decisions

### D1: Few-shot examples as YAML files appended to the instruction

**Choice:** Three YAML files under `agents/assistant/prompts/few-shot/`, each containing 4-6 examples. `load_prompt()` reads these files, formats them into a `## Classification Examples` section, and appends to the instruction string after skills.

**Files:**

| File | Classification | Examples |
|:--|:--|:--|
| `result-type.yaml` | Strength / Finding / Gap / Observation | Evidence patterns for each type, including edge cases: remediated Finding→Strength, active exception, mixed eval_results |
| `satisfaction.yaml` | Satisfied / Partially Satisfied / Not Satisfied / N/A | Cadence-complete vs cadence-gap scenarios, confidence thresholds, scope exclusions |
| `coverage-status.yaml` | Covered / Partially / Weakly / Not Covered / Unmapped | Strength+mapping combinations, multi-mapping resolution, no-mapping case |

**Structure per example:**

```yaml
- scenario: "Control BP-1, requirement BP-1.AR-1, target cluster-prod"
  evidence:
    eval_result: Passed
    compliance_status: Compliant
    confidence: High
    collected_at: "2026-01-15T00:00:00Z"
    cadence_gaps: 0
  classification: Strength
  determination: Satisfied
  reasoning: >
    Evidence exists within the audit window, eval_result is Passed,
    compliance_status is Compliant, confidence is High, no cadence gaps.
```

**Loading in `load_prompt()`:**

```python
def load_few_shot_examples() -> str:
    few_shot_dir = Path("/app/prompts/few-shot")
    if not few_shot_dir.exists():
        return ""
    parts = []
    for f in sorted(few_shot_dir.glob("*.yaml")):
        examples = yaml.safe_load(f.read_text())
        for ex in examples:
            parts.append(
                f"Scenario: {ex['scenario']}\n"
                f"Evidence: {ex['evidence']}\n"
                f"Classification: {ex['classification']}\n"
                f"Determination: {ex.get('determination', 'N/A')}\n"
                f"Reasoning: {ex['reasoning']}"
            )
    return "\n\n---\n\n".join(parts)
```

This adds ~10 lines to `main.py`. The existing `load_prompt()` structure is unchanged — examples are appended as a new `## Classification Examples` section after skills.

**Rationale:** Mapper's `prompts/base.yaml` uses this exact pattern — scenario + input + expected output + reasoning. The reasoning field teaches the model *why*, not just *what*. Anti-pattern examples (common misclassifications) are as important as positive examples.

**Alternative considered:** Embed examples inline in the `audit-methodology` skill. Rejected — examples are prompt engineering artifacts, not domain knowledge. They change with model behavior; skills change with methodology. Separate lifecycle.

**Alternative considered:** Full YAML merge semantics replacing `load_prompt()`. Deferred — adds runtime complexity (merge order, render function, deprecation of `prompt.md`) without proportional benefit at this stage. Revisit if prompt configs proliferate beyond the current structure.

### D2: Anti-pattern examples in each file

**Choice:** Each few-shot file includes at least one anti-pattern — a scenario where the obvious classification is wrong.

**Examples:**
- **Result type:** Evidence shows `eval_result: Failed` but `remediation_status: Success` within the audit window → Strength, not Finding. Without this example, the model defaults to Finding on any Failed result.
- **Satisfaction:** Control has `eval_result: Passed` but only one assessment cycle in a quarterly cadence → Partially Satisfied, not Satisfied. Without this, the model ignores cadence completeness.
- **Coverage:** Mapping `strength: 9` but AuditResult type is Gap → Not Covered, not Covered. High mapping strength doesn't override missing evidence.

**Rationale:** Mapper's `base.yaml` includes examples like "surface vocabulary overlap does NOT imply a genuine compliance relationship." Anti-patterns prevent the model from over-indexing on single features (e.g., mapping strength alone, eval_result alone).

### D3: AuditLog provenance columns

**Choice:** Add two nullable columns to the `audit_logs` table:

```sql
model Nullable(String),
prompt_version Nullable(String)
```

**Population path:**
- `model`: The gateway reads from the request body. The assistant's `after_agent_callback` can enrich the save request with `MODEL_NAME` (already an env var in `main.py`). If not provided, defaults to NULL.
- `prompt_version`: SHA256 hash of the full instruction string, computed once at startup in `main.py`. Stable per deployment — changes only when prompt content or skills change. Passed alongside `model` in the save request.

The gateway's `createAuditLogHandler` accepts these as optional fields. Missing fields default to NULL — backward-compatible with existing callers and the frontend `saveAuditLog` path.

**Rationale:** When a classification is disputed ("why was BP-3 marked as a Gap?"), the provenance columns answer: which model made the decision and which prompt version was active. Combined with the AuditLog `content` (which contains per-result reasoning), this provides a reproduction path. Deploying a new prompt version changes the hash, making it trivial to compare classification distributions before and after prompt changes.

**Alternative considered:** Store provenance in the AuditLog YAML itself. Rejected — provenance is about the production process, not the audit content. The Gemara `#AuditLog` schema doesn't have fields for model metadata, and extending it couples the schema to implementation details.

**Alternative considered:** Separate `audit_provenance` table. Rejected — one-to-one with `audit_logs`, adds a join for no benefit.

**Alternative considered:** Include `skills_loaded` array column. Deferred — all skills load on every turn today, so the value is always the same list. Revisit if selective skill loading is implemented.

## Risks / Trade-offs

| Risk | Mitigation |
|:--|:--|
| Few-shot examples may bias the model toward example patterns | Include anti-pattern examples that show common misclassifications. Review classification distribution in production — if >80% of results match a single example, the examples are too narrow. |
| Examples increase prompt token count (~300-400 tokens for 16 examples) | Marginal cost relative to the ~1500 lines of skills already loaded. Monitor via Vertex AI billing. |
| Provenance columns increase `audit_logs` row size | Nullable columns with no data cost ~1 byte per row in ClickHouse. `model` and `prompt_version` are short strings. Negligible. |
| `prompt_version` hash doesn't capture which skills were loaded | True, but all skills load every turn today. The hash of the full instruction string (which includes skills) implicitly captures skill content. If skills change, the hash changes. |
