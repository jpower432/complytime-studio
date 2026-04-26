---
name: studio-audit
description: Audit methodology, classification criteria, coverage mapping, and ClickHouse table reference
---

# Studio Audit

## Classification

| Type | Condition |
|:--|:--|
| Strength | eval_result = Passed, compliance_status = Compliant |
| Finding | eval_result = Failed, or cadence gaps detected |
| Gap | No evidence rows in audit window |
| Observation | eval_result = Needs Review, or mixed results |

Use most recent evidence per control+requirement. Enforcement with `remediation_status = Success` can convert Finding → Strength. Exception with `exception_active = true` converts Finding → annotated Strength.

## Satisfaction

| Determination | Condition |
|:--|:--|
| Satisfied | Evidence complete, current, confidence Medium/High, no cadence gaps |
| Partially Satisfied | Incomplete evidence, missing cycles, Low confidence, mixed results |
| Not Satisfied | Failed eval_result, critical cadence gaps without remediation |
| Not Applicable | Control scoped out for this target |

Never mark Satisfied without evidence. Absence = Gap.

## Cadence

Map `Policy.adherence.assessment-plans[].frequency` to cycle length (daily=1d, weekly=7d, monthly=30d, quarterly=90d, annually=365d). Expected cycles = floor((end - start) / cycle_length). Missing cycles are Findings.

## Coverage Mapping

When `mapping_documents` exist for the policy, join AuditResults with mapping entries:

| AuditResult | Strength 8-10 | 5-7 | 1-4 |
|:--|:--|:--|:--|
| Strength | Covered | Partially Covered | Weakly Covered |
| Finding | Not Covered | Not Covered | Not Covered |
| Gap | Not Covered | Not Covered | Not Covered |
| Observation | Needs Review | Needs Review | Needs Review |

Multiple controls mapping to the same external entry: use strongest coverage. No mapping documents = skip cross-framework analysis.

## MCP Tools

**validate_gemara_artifact**: `artifact_content` (YAML string), `definition` (e.g. `#AuditLog`), `version` (optional)
**migrate_gemara_artifact**: `artifact_content` (YAML string), `artifact_type` (optional), `gemara_version` (optional)

## ClickHouse Tables

Database: `default`. Query via `run_select_query`. Use `DESCRIBE TABLE <name>` for column types.

```
evidence: evidence_id, target_id, target_name, target_type, target_env, engine_name, engine_version, rule_id, rule_name, eval_result, eval_message, policy_id, control_id, control_catalog_id, control_category, control_applicability, requirement_id, plan_id, confidence, compliance_status, risk_level, requirements, remediation_action, remediation_status, remediation_desc, exception_id, exception_active, enrichment_status, collected_at, ingested_at, attestation_ref
policies: policy_id, title, version, oci_reference, content, imported_at, imported_by
noncompliant_evidence: (same schema as evidence, pre-filtered to failing/non-compliant rows)
evidence_assessments: evidence_id, policy_id, plan_id, classification, reason, assessed_at, assessed_by
mapping_documents: mapping_id, policy_id, framework, content, imported_at
mapping_entries: mapping_id, policy_id, control_id, requirement_id, framework, reference, strength, confidence, imported_at
catalogs: catalog_id, catalog_type, title, content, policy_id, imported_at
controls: catalog_id, control_id, title, objective, group_id, state, policy_id, imported_at
assessment_requirements: catalog_id, control_id, requirement_id, text, applicability, recommendation, state, imported_at
control_threats: catalog_id, control_id, threat_reference_id, threat_entry_id, imported_at
threats: catalog_id, threat_id, title, description, group_id, policy_id, imported_at
risks: catalog_id, risk_id, title, description, severity, group_id, impact, policy_id, imported_at
risk_threats: catalog_id, risk_id, threat_reference_id, threat_entry_id, imported_at
audit_logs: audit_id, policy_id, audit_start, audit_end, framework, created_at, created_by, content, summary, model, prompt_version
draft_audit_logs: draft_id, policy_id, audit_start, audit_end, framework, created_at, status, content, summary, agent_reasoning, model, prompt_version, reviewed_by, promoted_at, reviewer_edits
```

## Risk Severity Queries

Derive risk severity for failing evidence via threat linkage:

```sql
SELECT e.control_id, e.eval_result, r.risk_id, r.title AS risk_title, r.severity
FROM evidence e
INNER JOIN control_threats ct ON e.control_id = ct.control_id
INNER JOIN risk_threats rt ON ct.threat_entry_id = rt.threat_entry_id
INNER JOIN risks r ON r.risk_id = rt.risk_id AND r.catalog_id = rt.catalog_id
WHERE e.policy_id = ? AND e.eval_result = 'Failed'
```

Risk exposure summary (counts by severity):

```sql
SELECT r.severity, count(DISTINCT r.risk_id) AS risk_count, count(DISTINCT e.control_id) AS affected_controls
FROM evidence e
INNER JOIN control_threats ct ON e.control_id = ct.control_id
INNER JOIN risk_threats rt ON ct.threat_entry_id = rt.threat_entry_id
INNER JOIN risks r ON r.risk_id = rt.risk_id AND r.catalog_id = rt.catalog_id
WHERE e.policy_id = ? AND e.eval_result = 'Failed'
GROUP BY r.severity
```

Unmitigated risks (threats with no control):

```sql
SELECT r.risk_id, r.title, r.severity
FROM risk_threats rt
INNER JOIN risks r ON r.risk_id = rt.risk_id AND r.catalog_id = rt.catalog_id
LEFT JOIN control_threats ct ON rt.threat_entry_id = ct.threat_entry_id
WHERE ct.control_id IS NULL
```
