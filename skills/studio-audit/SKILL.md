---
name: studio-audit
description: Audit methodology, classification criteria, coverage mapping, and PostgreSQL schema reference
---

# Studio Audit

## Classification

| Type | Condition |
|:--|:--|
| Strength | eval_result = Passed, compliance_status = Compliant |
| Finding | eval_result = Failed, or cadence gaps detected |
| Gap | No evidence rows in audit window |
| Observation | eval_result = Needs Review, or mixed results |

Use most recent evidence per control+requirement. Enforcement with `remediation_status = Success` can convert Finding -> Strength. Exception with `exception_active = true` converts Finding -> annotated Strength.

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
**query_database**: `query` (string) — execute a read-only SELECT against PostgreSQL
**get_schema_info**: `schema_name` (optional), `table_name` (optional) — introspect database tables and columns

## Workbench posture list vs. SQL

The workbench calls `GET /api/posture` with optional `start` and `end` query parameters to limit rows to evidence whose `collected_at` falls in that window (omitting both parameters includes all evidence). When you run ad-hoc SQL that should match the on-screen posture numbers, apply the same `collected_at` range the user selected (including workbench time presets: 7d, 30d, 90d, or all-time).

## PostgreSQL Tables

All data lives in PostgreSQL. Use `query_database` for SQL access and `get_schema_info` to discover tables.

| Table | Key Columns | Purpose |
|:--|:--|:--|
| `policies` | policy_id, title, content (YAML), oci_reference | Imported L3 policies |
| `evidence` | evidence_id, target_id, policy_id, control_id, requirement_id, eval_result, compliance_status, collected_at | Flattened evaluation evidence |
| `mapping_documents` | mapping_id, policy_id, framework, content | Cross-framework mappings |
| `mapping_entries` | mapping_id, policy_id, control_id, requirement_id, framework, reference, strength | Individual mapping links |
| `catalogs` | catalog_id, catalog_type, title, policy_id | Control/threat/risk catalogs |
| `controls` | catalog_id, control_id, title, objective, policy_id | Individual controls |
| `assessment_requirements` | catalog_id, control_id, requirement_id, text | Assessment criteria |
| `threats` | catalog_id, threat_id, title, description, policy_id | Threat entries |
| `risks` | catalog_id, risk_id, title, severity, impact, policy_id | Risk entries |
| `control_threats` | catalog_id, control_id, threat_reference_id, threat_entry_id | Control-threat links |
| `risk_threats` | catalog_id, risk_id, threat_reference_id, threat_entry_id | Risk-threat links |
| `audit_logs` | audit_id, policy_id, audit_start, audit_end, content, summary | Promoted audit logs |
| `draft_audit_logs` | draft_id, policy_id, status, content, agent_reasoning | Draft audit logs |
| `certifications` | id, evidence_id, certifier, result, reason | Evidence certification verdicts |
| `evidence_assessments` | id, evidence_id, policy_id, plan_id, classification, reason | Posture check results |
| `programs` | id, name, description, owner, policy_ids | Compliance programs |

## Example Queries

```sql
-- All evidence for a policy within an audit window
SELECT evidence_id, target_id, control_id, eval_result, collected_at
FROM evidence
WHERE policy_id = 'ampel-branch-protection'
  AND collected_at BETWEEN '2026-01-01' AND '2026-03-31'
ORDER BY collected_at DESC;

-- Risk exposure by severity
SELECT r.severity, COUNT(*) AS risk_count
FROM risks r
GROUP BY r.severity
ORDER BY CASE r.severity
  WHEN 'Critical' THEN 1 WHEN 'High' THEN 2
  WHEN 'Medium' THEN 3 WHEN 'Low' THEN 4 ELSE 5
END;

-- Failing evidence with threat chain
SELECT e.evidence_id, e.control_id, e.eval_result,
       t.title AS threat, r.title AS risk, r.severity
FROM evidence e
JOIN control_threats ct ON ct.control_id = e.control_id
JOIN threats t ON t.threat_id = ct.threat_entry_id
JOIN risk_threats rt ON rt.threat_entry_id = t.threat_id
JOIN risks r ON r.risk_id = rt.risk_id
WHERE e.policy_id = 'ampel-branch-protection'
  AND e.eval_result = 'Failed';

-- Coverage matrix: controls mapped to external frameworks
SELECT me.control_id, me.framework, me.reference, me.strength
FROM mapping_entries me
WHERE me.policy_id = 'ampel-branch-protection'
ORDER BY me.framework, me.reference;
```
