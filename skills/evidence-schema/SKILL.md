---
name: evidence-schema
description: ClickHouse table schemas, enum values, and query patterns for compliance evidence and audit data
---

# Evidence Schema

All data is in the `default` database, queried via `run_select_query` through the clickhouse-mcp server. Always use `database: "default"` when calling `list_tables` or `list_databases`.

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

### mapping_entries

Structured mapping entries parsed from mapping document YAML at import time. Each row is one control-to-framework-objective relationship. Enables SQL joins between evidence and framework objectives without YAML parsing.

| Column | Type | Description |
|:--|:--|:--|
| mapping_id | String | Parent mapping_documents row |
| policy_id | String | Associated policy |
| control_id | String | Source control entry ID (e.g., BP-1) |
| requirement_id | String | Mapping entry ID from the Gemara MappingDocument |
| framework | String | Target framework (e.g., SOC 2) |
| reference | String | Framework objective (e.g., CC8.1) |
| strength | UInt8 | Author's estimate of how completely the control satisfies the objective (1-10, 0=unset) |
| confidence | String | Confidence level (Undetermined, Low, Medium, High) |
| imported_at | DateTime64(3) | Import timestamp |

### catalogs

Raw catalog artifacts (ControlCatalog, ThreatCatalog) imported via `/api/catalogs/import`.

| Column | Type | Description |
|:--|:--|:--|
| catalog_id | String | Catalog identifier (from metadata.id) |
| catalog_type | LowCardinality(String) | Artifact type: ControlCatalog, ThreatCatalog |
| title | String | Catalog title |
| content | String | Full catalog YAML |
| policy_id | String | Policy that imported this catalog (provenance) |
| imported_at | DateTime64(3) | Import timestamp |

### controls

Parsed L2 ControlCatalog entries. One row per control.

| Column | Type | Description |
|:--|:--|:--|
| catalog_id | String | Parent ControlCatalog ID |
| control_id | String | Control identifier |
| title | String | Control title |
| objective | String | Control objective statement |
| group_id | String | Catalog group this control belongs to |
| state | LowCardinality(String) | Lifecycle state: Active, Draft, Deprecated, Retired |
| policy_id | String | Policy that imported the catalog (provenance) |
| imported_at | DateTime64(3) | Import timestamp |

### assessment_requirements

Parsed L2 assessment requirements. One row per requirement, linked to its parent control.

| Column | Type | Description |
|:--|:--|:--|
| catalog_id | String | Parent ControlCatalog ID |
| control_id | String | Parent control ID |
| requirement_id | String | Requirement identifier |
| text | String | Requirement text (MUST condition) |
| applicability | Array(String) | Applicability tags |
| recommendation | String | Non-binding recommendation |
| state | LowCardinality(String) | Lifecycle state |
| imported_at | DateTime64(3) | Import timestamp |

### control_threats

Junction table linking controls to threats via `Control.threats` cross-references.

| Column | Type | Description |
|:--|:--|:--|
| catalog_id | String | Parent ControlCatalog ID |
| control_id | String | Control that mitigates the threat |
| threat_reference_id | String | Reference to the ThreatCatalog mapping reference |
| threat_entry_id | String | Specific threat ID within the referenced catalog |
| imported_at | DateTime64(3) | Import timestamp |

### threats

Parsed L2 ThreatCatalog entries. One row per threat.

| Column | Type | Description |
|:--|:--|:--|
| catalog_id | String | Parent ThreatCatalog ID |
| threat_id | String | Threat identifier |
| title | String | Threat title |
| description | String | Threat description |
| group_id | String | Catalog group this threat belongs to |
| policy_id | String | Policy that imported the catalog (provenance) |
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
| model | Nullable(String) | LLM model that produced the artifact |
| prompt_version | Nullable(String) | SHA256 prefix of the instruction string active at creation |

## Query Patterns

**IMPORTANT:** The queries below use `<PLACEHOLDER>` syntax for values you must substitute. Replace each `<PLACEHOLDER>` with the actual value as a quoted string literal. Do NOT send curly braces `{...}` to ClickHouse — ClickHouse interprets `{name}` as a query parameter and will error with "Context variable not found."

### Target inventory

Discover all targets with evidence for a policy within the audit window:

```sql
SELECT DISTINCT target_id, target_name, target_type,
       min(collected_at) AS earliest, max(collected_at) AS latest,
       count(*) AS evidence_count
FROM evidence
WHERE policy_id = '<POLICY_ID>'
  AND collected_at BETWEEN '<START_DATE>' AND '<END_DATE>'
GROUP BY target_id, target_name, target_type
ORDER BY target_id
```

### Per-target evidence

Retrieve evidence for a specific target, ordered for assessment:

```sql
SELECT *
FROM evidence
WHERE policy_id = '<POLICY_ID>'
  AND target_id = '<TARGET_ID>'
  AND collected_at BETWEEN '<START_DATE>' AND '<END_DATE>'
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
WHERE policy_id = '<POLICY_ID>'
  AND target_id = '<TARGET_ID>'
  AND collected_at BETWEEN '<START_DATE>' AND '<END_DATE>'
GROUP BY control_id, requirement_id
ORDER BY control_id, requirement_id
```

### Policy retrieval

Load the full Policy content for parsing:

```sql
SELECT policy_id, title, version, content
FROM policies
WHERE policy_id = '<POLICY_ID>'
ORDER BY imported_at DESC
LIMIT 1
```

### Mapping documents for a policy

```sql
SELECT mapping_id, framework, content
FROM mapping_documents
WHERE policy_id = '<POLICY_ID>'
ORDER BY framework
```

### Impact / blast radius

Join evidence failures against structured mapping entries to determine which framework objectives are affected. This is the primary query for answering "which certifications or ATOs are potentially affected by this failure?"

```sql
SELECT e.control_id, e.target_name, e.eval_result,
       m.framework, m.reference, m.strength, m.confidence
FROM evidence e
JOIN mapping_entries m
  ON e.policy_id = m.policy_id AND e.control_id = m.control_id
WHERE e.policy_id = '<POLICY_ID>'
  AND e.eval_result IN ('Failed', 'Not Run')
  AND e.collected_at BETWEEN '<START_DATE>' AND '<END_DATE>'
ORDER BY m.framework, m.reference, m.strength DESC
```

### Threat impact traversal

Find all evidence for controls that mitigate a specific threat (threat → control → evidence):

```sql
SELECT e.target_id, e.control_id, e.eval_result, e.collected_at
FROM control_threats ct
JOIN evidence e
  ON e.control_id = ct.control_id
WHERE ct.threat_entry_id = '<THREAT_ID>'
  AND e.policy_id = '<POLICY_ID>'
ORDER BY e.collected_at DESC
```

### Coverage completeness

Find controls with no evidence (gap detection):

```sql
SELECT c.control_id, c.title, c.objective
FROM controls c
LEFT JOIN evidence e
  ON e.control_id = c.control_id AND e.policy_id = '<POLICY_ID>'
WHERE c.catalog_id = '<CATALOG_ID>'
  AND e.control_id IS NULL
ORDER BY c.control_id
```

### Requirement text enrichment

Enrich evidence rows with the human-readable assessment requirement text:

```sql
SELECT e.control_id, e.requirement_id, e.eval_result,
       ar.text AS requirement_text, ar.recommendation
FROM evidence e
JOIN assessment_requirements ar
  ON ar.control_id = e.control_id
  AND ar.requirement_id = e.requirement_id
WHERE e.policy_id = '<POLICY_ID>'
ORDER BY e.control_id, e.requirement_id
```

### Framework-to-threat traversal

Discover which threats are addressed by controls mapped to a framework (framework → mapping → control → threat):

```sql
SELECT DISTINCT t.threat_id, t.title, t.description
FROM mapping_entries me
JOIN controls c
  ON c.control_id = me.control_id
JOIN control_threats ct
  ON ct.control_id = c.control_id AND ct.catalog_id = c.catalog_id
JOIN threats t
  ON t.threat_id = ct.threat_entry_id
WHERE me.framework = '<FRAMEWORK>'
  AND me.policy_id = '<POLICY_ID>'
ORDER BY t.threat_id
```

### Impact aggregation by framework objective

Summarize the blast radius by grouping failed controls per framework objective:

```sql
SELECT m.framework, m.reference,
       count(DISTINCT e.control_id) AS failed_controls,
       count(DISTINCT e.target_name) AS affected_targets,
       max(m.strength) AS max_strength,
       groupArray(DISTINCT e.control_id) AS control_ids
FROM evidence e
JOIN mapping_entries m
  ON e.policy_id = m.policy_id AND e.control_id = m.control_id
WHERE e.policy_id = '<POLICY_ID>'
  AND e.eval_result IN ('Failed', 'Not Run')
  AND e.collected_at BETWEEN '<START_DATE>' AND '<END_DATE>'
GROUP BY m.framework, m.reference
ORDER BY max_strength DESC, m.framework, m.reference
```
