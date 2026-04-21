---
name: audit-methodology
description: Assessment cadence rules, frequency mapping, finding classification, and satisfaction determination for audit preparation
---

# Audit Methodology

## Assessment Cadence

Policies define assessment frequency in `Policy.adherence.assessment-plans[].frequency`. Map frequency to expected assessment cycles within the audit window:

| Frequency | Cycle length |
|:--|:--|
| daily | 1 day |
| weekly | 7 days |
| monthly | 30 days |
| quarterly | 90 days |
| annually | 365 days |

**Expected cycles** = floor((audit_end - audit_start) / cycle_length).

Query actual assessment timestamps from ClickHouse. Any expected cycle without evidence is a **cadence gap**. Cadence gaps are classified as Findings — the target failed to meet the policy's continuous compliance requirement. Document the specific date ranges where evidence is missing.

## Classification Criteria

Every criteria entry (control + assessment requirement from the Policy) is classified into one of four types based on evidence:

| Type | Condition | Meaning |
|:--|:--|:--|
| Strength | Evidence exists, eval_result = Passed, compliance_status = Compliant | Criteria is met — evidence supports compliance |
| Finding | Evidence exists but eval_result = Failed, or cadence gaps detected | Criteria is not met — specific non-compliance identified |
| Gap | No evidence rows within the audit window for this criteria entry | Criteria is untested — no data to evaluate |
| Observation | Evidence exists but eval_result = Needs Review, or mixed results | Criteria needs human judgment — evidence is ambiguous |

**Rules:**
- Use the most recent evidence rows when multiple exist for the same requirement
- Evaluation-only rows (NULL remediation) and enforcement rows are both valid evidence
- Enforcement rows with `remediation_status = Success` may convert a Finding into a Strength if the remediation resolved the issue within the audit window
- An exception with `exception_active = true` converts a Finding into an annotated Strength (note the exception)

## Satisfaction Determination

Adapted from assessment practice. Each criteria entry receives an explicit determination:

| Determination | Criteria |
|:--|:--|
| Satisfied | Evidence is complete, current, and supports compliance. No cadence gaps. Confidence is Medium or High. |
| Partially Satisfied | Evidence exists but is incomplete — some assessment cycles missing, or confidence is Low, or mixed eval_results across the window |
| Not Satisfied | Evidence shows non-compliance (Failed eval_result), or critical cadence gaps with no remediation |
| Not Applicable | Control is scoped out for this target (control_applicability does not match target_type/target_env) |

**Never mark a criteria entry as Satisfied without evidence.** Absence of evidence is a Gap, not implicit compliance.

## Citation Quality

When referencing evidence in AuditResults, ground every claim in a specific evidence record:

| Confidence | Criteria |
|:--|:--|
| High | Specific evidence_id cited, eval_result is unambiguous, collected within audit window |
| Medium | Evidence exists but from a single assessment cycle, or confidence field is Low |
| Low | Evidence is inferred from related controls, or collected outside the audit window |
| Not Found | No evidence record can be located — classify as Gap |

## AuditResult Structure

Each AuditResult in the AuditLog maps to one criteria entry and includes:

- `criteria-reference`: The control + assessment requirement being assessed
- `type`: Strength, Finding, Gap, or Observation
- `description`: Factual summary of what the evidence shows
- `recommendations`: Actionable items (for Findings and Gaps)
- Coverage and cadence data supporting the classification

**Completeness is mandatory.** Every criteria entry from the Policy MUST have a corresponding AuditResult per target. Missing AuditResults are a validation error.
