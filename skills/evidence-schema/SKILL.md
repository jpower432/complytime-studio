---
name: evidence-schema
description: ClickHouse table schemas, enum values, and query patterns for compliance evidence and audit data
---

# Evidence Schema

All data is queried via `run_select_query` through the clickhouse-mcp server.

## Tables

### evidence

Pre-evaluated compliance evidence. Each row is one rule evaluation against one target.

| Column | Type | Description |
|:--|:--|:--|
| evidence_id | String | Unique evidence record ID |
| target_id | String | Resource being evaluated |
| target_name | Nullable(String) | Human-readable target name |
| target_type | Nullable(String) | Target type (e.g., cluster, node, image) |
| target_env | Nullable(String) | Deployment environment |
| engine_name | Nullable(String) | Evaluation engine name |
| engine_version | Nullable(String) | Evaluation engine version |
| rule_id | String | Rule that was evaluated |
| rule_name | Nullable(String) | Human-readable rule name |
| rule_uri | Nullable(String) | Rule reference URI |
| eval_result | Enum | Evaluation outcome |
| eval_message | Nullable(String) | Evaluation detail message |
| policy_id | Nullable(String) | Policy this evidence maps to |
| control_id | Nullable(String) | Control being assessed |
| control_catalog_id | Nullable(String) | Source control catalog |
| control_category | Nullable(String) | Control category/family |
| control_applicability | Array(String) | Applicability tags |
| requirement_id | Nullable(String) | Specific assessment requirement |
| plan_id | Nullable(String) | Assessment plan ID |
| confidence | Nullable(Enum) | Confidence in the evaluation |
| steps_executed | Nullable(UInt16) | Number of assessment steps run |
| compliance_status | Enum | Overall compliance determination |
| risk_level | Nullable(Enum) | Risk severity |
| frameworks | Array(String) | Mapped compliance frameworks |
| requirements | Array(String) | Mapped framework requirements |
| remediation_action | Nullable(Enum) | Action taken (enforcement rows only) |
| remediation_status | Nullable(Enum) | Remediation outcome (enforcement rows only) |
| remediation_desc | Nullable(String) | Remediation description |
| exception_id | Nullable(String) | Exception reference if waived |
| exception_active | Nullable(Bool) | Whether exception is active |
| enrichment_status | Enum | Policy-mapping enrichment status |
| collected_at | DateTime64(3) | When evidence was collected |
| ingested_at | DateTime64(3) | When evidence was stored (auto) |

**Evaluation-only rows** have NULL remediation columns. **Enforcement rows** have `remediation_action` and `remediation_status` populated.

**Enum values:**

| Column | Values |
|:--|:--|
| eval_result | Not Run, Passed, Failed, Needs Review, Not Applicable, Unknown |
| confidence | Undetermined, Low, Medium, High |
| compliance_status | Compliant, Non-Compliant, Exempt, Not Applicable, Unknown |
| risk_level | Critical, High, Medium, Low, Informational |
| remediation_action | Block, Allow, Remediate, Waive, Notify, Unknown |
| remediation_status | Success, Fail, Skipped, Unknown |
| enrichment_status | Success, Unmapped, Partial, Unknown, Skipped |

### policies

Imported L3 Policy artifacts.

| Column | Type | Description |
|:--|:--|:--|
| policy_id | String | Policy identifier |
| title | String | Policy title |
| version | Nullable(String) | Policy version |
| oci_reference | String | OCI bundle reference |
| content | String | Full Policy YAML |
| imported_at | DateTime64(3) | Import timestamp |
| imported_by | Nullable(String) | Importing user |

### mapping_documents

Cross-framework mapping documents linking internal criteria to external frameworks.

| Column | Type | Description |
|:--|:--|:--|
| mapping_id | String | Mapping document ID |
| policy_id | String | Associated policy |
| framework | String | Target framework (e.g., SOC 2, ISO 27001) |
| content | String | Full MappingDocument YAML |
| imported_at | DateTime64(3) | Import timestamp |

### audit_logs

Completed L7 AuditLog artifacts produced by the assistant.

| Column | Type | Description |
|:--|:--|:--|
| audit_id | String | Audit identifier |
| policy_id | String | Policy audited |
| audit_start | DateTime64(3) | Audit period start |
| audit_end | DateTime64(3) | Audit period end |
| framework | Nullable(String) | Framework scope |
| created_at | DateTime64(3) | Creation timestamp |
| created_by | Nullable(String) | Creating user |
| content | String | Full AuditLog YAML |
| summary | String | Audit summary text |

## Query Patterns

### Target inventory

Discover all targets with evidence for a policy within the audit window:

```sql
SELECT DISTINCT target_id, target_name, target_type,
       min(collected_at) AS earliest, max(collected_at) AS latest,
       count(*) AS evidence_count
FROM evidence
WHERE policy_id = '{policy_id}'
  AND collected_at BETWEEN '{start}' AND '{end}'
GROUP BY target_id, target_name, target_type
ORDER BY target_id
```

### Per-target evidence

Retrieve evidence for a specific target, ordered for assessment:

```sql
SELECT *
FROM evidence
WHERE policy_id = '{policy_id}'
  AND target_id = '{target_id}'
  AND collected_at BETWEEN '{start}' AND '{end}'
ORDER BY control_id, requirement_id, collected_at DESC
```

Use the most recent row when multiple exist for the same control + requirement.

### Assessment cadence validation

Count assessment timestamps per criteria entry to detect missing cycles:

```sql
SELECT control_id, requirement_id,
       count(DISTINCT toDate(collected_at)) AS assessment_days,
       min(collected_at) AS first_seen, max(collected_at) AS last_seen
FROM evidence
WHERE policy_id = '{policy_id}'
  AND target_id = '{target_id}'
  AND collected_at BETWEEN '{start}' AND '{end}'
GROUP BY control_id, requirement_id
ORDER BY control_id, requirement_id
```

### Policy retrieval

Load the full Policy content for parsing:

```sql
SELECT policy_id, title, version, content
FROM policies
WHERE policy_id = '{policy_id}'
ORDER BY imported_at DESC
LIMIT 1
```

### Mapping documents for a policy

```sql
SELECT mapping_id, framework, content
FROM mapping_documents
WHERE policy_id = '{policy_id}'
ORDER BY framework
```
