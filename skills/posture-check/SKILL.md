---
name: posture-check
description: Pre-audit readiness check — validates evidence stream against Policy assessment plans for cadence, provenance, method, evidence fitness, and result quality
---

# Posture Check

Evaluate compliance readiness by joining a Policy's assessment plans against the evidence stream in ClickHouse. Returns a per-plan, per-target readiness table and emits a structured EvidenceAssessment artifact for Gateway persistence.

## Assessment Plan Extraction

Parse the `policies.content` YAML column to extract `adherence.assessment-plans[]`. Each plan has:

| Field | YAML Path | Use |
|:--|:--|:--|
| Plan ID | `assessment-plans[].id` | Row key in readiness table |
| Requirement ID | `assessment-plans[].requirement-id` | Join to `evidence.requirement_id` |
| Frequency | `assessment-plans[].frequency` | Cadence window calculation |
| Evaluation Methods | `assessment-plans[].evaluation-methods[]` | Method/mode/executor validation |
| Executor ID | `assessment-plans[].evaluation-methods[].executor.id` | Provenance check vs `evidence.engine_name` |
| Executor Type | `assessment-plans[].evaluation-methods[].executor.type` | Human vs Software |
| Mode | `assessment-plans[].evaluation-methods[].mode` | Manual vs Automated |
| Type | `assessment-plans[].evaluation-methods[].type` | Intent vs Behavioral |
| Evidence Reqs | `assessment-plans[].evidence-requirements` | Semantic fitness comparison |

If the Policy has no `adherence.assessment-plans`, report "Policy has no assessment plans defined" and halt.

## Evidence Query

For each assessment plan, query the `evidence` table via `run_select_query`. Filter by `policy_id` and `control_id` within the frequency-derived window. Order by `collected_at DESC` to get the most recent evidence first.

Columns needed: `evidence_id`, `engine_name`, `engine_version`, `eval_result`, `collected_at`, `confidence`, `control_id`, `target_id`, `target_name`, `attestation_ref`.

IMPORTANT: Use literal string values in SQL queries, not template variables like `plan_id:String`. The `plan_id` and `requirement_id` columns are often NULL. Always filter by `control_id` first. Only add `AND plan_id = 'value'` or `AND requirement_id = 'value'` when you have a known non-empty value to match.

## Provenance Validation

Two modes, selected by `attestation_ref` presence:

### Mode A: Cryptographic verification (attestation_ref present)

When `evidence.attestation_ref` is not NULL, invoke the attestation-verification skill to verify the chain. The verdict determines the provenance result:

| Verdict | Provenance Result |
|:--|:--|
| CHAIN VERIFIED | Pass — note "cryptographically verified" |
| BROKEN CHAIN | **Wrong Source** — include the chain failure reason |
| REGISTRY UNAVAILABLE | Fall back to Mode B with note |

### Mode B: String comparison (attestation_ref NULL or fallback)

Compare `evidence.engine_name` against the plan's `evaluation-methods[].executor.id`:

| Evidence `engine_name` | Plan `executor.id` | Result |
|:--|:--|:--|
| Matches | Defined | Pass |
| Does not match | Defined | **Wrong Source** — "Expected: <executor.id>, Got: <engine_name>" |
| NULL | Defined | **Wrong Source** — provenance cannot be verified |
| Any | Not defined | Skip check — plan does not constrain executor |

When `executor.version` is defined, also compare `evidence.engine_version`. Version mismatch alone does not trigger Wrong Source but is noted.

## Method/Mode Validation

Compare evidence collection context against the plan's `evaluation-methods[]`:

| Check | Evidence Signal | Plan Field | Mismatch Result |
|:--|:--|:--|:--|
| Mode | OTel collector path → Automated; REST upload → Manual | `mode` | **Wrong Method** — "Plan requires <mode>, evidence was <actual>" |
| Type | Evidence metadata indicates Intent or Behavioral | `type` | **Wrong Method** — "Plan requires <type>, evidence is <actual>" |

If the plan's `evaluation-methods[]` does not specify mode or type, skip the corresponding check.

## Evidence Fitness

Compare evidence content against the plan's `evidence-requirements` field. This is a semantic comparison — use reasoning to determine whether the evidence satisfies the described requirement.

| Evidence Content | Plan `evidence-requirements` | Result |
|:--|:--|:--|
| Matches described requirement | Defined | Pass |
| Does not match described requirement | Defined | **Unfit Evidence** — explain mismatch |
| Any | Not defined | Skip check |

Example: plan requires "Firewall rule export showing ingress/egress policies" but evidence is a Kyverno pod security report → Unfit Evidence.

## Cadence

| Frequency | Cycle Length |
|:--|:--|
| daily | 1 day |
| weekly | 7 days |
| monthly | 30 days |
| quarterly | 90 days |
| annually | 365 days |

Expected cycles within audit window = floor((window_end - window_start) / cycle_length). Compare against actual distinct collection periods in evidence. Missing cycles classify the plan as Stale.

## Classification

Each assessment plan is classified into one of seven states. When multiple conditions apply, use the highest-priority (worst) state.

| Priority | State | Condition |
|:--|:--|:--|
| 1 (worst) | **No Evidence** | No evidence rows for this plan's `requirement_id` within the audit window |
| 2 | **Wrong Source** | Evidence exists but provenance check failed (engine_name mismatch or broken attestation chain) |
| 3 | **Wrong Method** | Evidence exists, correct source, but method/mode does not match plan's evaluation-methods |
| 4 | **Unfit Evidence** | Evidence exists, correct source and method, but content does not satisfy evidence-requirements |
| 5 | **Stale** | Evidence exists from correct source/method/fitness but most recent is outside the current frequency window |
| 6 | **Failing** | Evidence exists, correct source/method/fitness, on cadence, but latest `eval_result` = Failed or Needs Review |
| 7 (best) | **Healthy** | Evidence exists, correct source/method/fitness, on cadence, latest `eval_result` = Passed |

## Readiness Table Format

Return one table per target:

```
Policy: <title> (<policy_id>)
Target: <target_name> (<target_id>)
Window: <window_start> — <window_end>

| Plan   | Frequency | Last Evidence | Provenance         | Method | Result | Status         |
|:-------|:----------|:--------------|:-------------------|:-------|:-------|:---------------|
| AP-01  | weekly    | 2d ago        | ✓ verified (chain) | ✓ Auto | Passed | Healthy        |
| AP-02  | quarterly | 45d ago       | ✓ nessus (name)    | ✓ Auto | Failed | Failing        |
| AP-03  | quarterly | 190d ago      | ✗ qualys≠nessus    | —      | Passed | Wrong Source   |
| AP-04  | monthly   | 3d ago        | ✓ opa (name)       | ✗ Manual≠Auto | Passed | Wrong Method |
| AP-05  | monthly   | —             | —                  | —      | —      | No Evidence    |

Summary: 1/5 plans healthy. 1 failing, 1 wrong source, 1 wrong method, 1 no evidence.
```

## EvidenceAssessment Artifact

After presenting the readiness table, emit a structured EvidenceAssessment artifact as an A2A artifact part with mimeType `application/yaml`:

```yaml
type: EvidenceAssessment
policy_id: <policy_id>
assessed_at: "{ISO-8601 now}"
assessed_by: "<model>/<prompt_version>"
assessments:
  - evidence_id: ev-123
    plan_id: AP-01
    classification: Healthy
    reason: "Evidence current, actor verified via attestation chain, result Passed"
  - evidence_id: ev-456
    plan_id: AP-02
    classification: Failing
    reason: "Evidence current, source matches (nessus), but eval_result = Failed"
```

The Gateway auto-persists this to the `evidence_assessments` table. One entry per evidence row assessed.

For plans classified as No Evidence, omit from assessments (no evidence_id to reference).

## ClickHouse Tables

Database: `default`. Query via `run_select_query`. Use `DESCRIBE TABLE <name>` for column types.

```
evidence: evidence_id, target_id, target_name, target_type, target_env, engine_name, engine_version, rule_id, rule_name, eval_result, eval_message, policy_id, control_id, control_catalog_id, control_category, control_applicability, requirement_id, plan_id, confidence, compliance_status, risk_level, requirements, remediation_action, remediation_status, remediation_desc, exception_id, exception_active, enrichment_status, collected_at, ingested_at, attestation_ref
policies: policy_id, title, version, oci_reference, content, imported_at, imported_by
noncompliant_evidence: (same schema as evidence, pre-filtered to failing/non-compliant rows)
evidence_assessments: evidence_id, policy_id, plan_id, classification, reason, assessed_at, assessed_by
```
