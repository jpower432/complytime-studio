# QE Test Instructions: gap-analyst-audit-prep

**Change:** Evolve gap-analyst from single-target evidence synthesizer to combined audit preparation assistant with multi-target inventory derivation, cadence validation, and cross-framework coverage analysis via MappingDocuments.

**Prerequisites:**
- ClickHouse deployed with `evaluation_logs` and `enforcement_actions` tables populated
- At least two distinct `target_id` values with evidence for the same `policy_id`
- A valid Gemara Policy YAML with `adherence.assessment-plans[].frequency` defined
- At least one valid Gemara MappingDocument mapping internal controls to an external framework
- gemara-mcp and clickhouse-mcp servers available

---

## Happy Path

### HP-1: Agent picker shows updated description

| Step | Action | Expected |
|:-----|:-------|:---------|
| 1 | Open workbench, click "+ New Job" | Agent picker lists `studio-gap-analyst` |
| 2 | Inspect the agent card | Description reads "Combined audit preparation" (not "Evidence synthesis from pre-evaluated L5/L6 data") |
| 3 | Inspect A2A skill | Skill name is "Combined Audit Preparation", tags include `combined-audit` |

### HP-2: Phase 1 — Scope confirmation and inventory derivation

| Step | Action | Expected |
|:-----|:-------|:---------|
| 1 | Select gap-analyst, provide Policy YAML + 2 MappingDocuments + audit timeline (e.g., "Jan 1 – Mar 31 2026") | Agent confirms scope: policy, timeline, frameworks listed in a summary table |
| 2 | Wait for inventory query | Agent queries ClickHouse (`SELECT DISTINCT target_id ...`) and presents an inventory table with target name, type, eval count, and evidence window |
| 3 | Confirm targets (e.g., "Audit prod-cluster-east and prod-cluster-west") | Agent acknowledges selection, proceeds to Phase 2 |

### HP-3: Phase 2 — Evidence assessment per target

| Step | Action | Expected |
|:-----|:-------|:---------|
| 1 | Wait for evidence queries | Agent queries `evaluation_logs` and `enforcement_actions` per confirmed target |
| 2 | Observe cadence validation | Agent computes expected assessment cycles from policy frequency, identifies any missing cycles |
| 3 | Observe classification | Agent classifies each criteria entry (Strength, Finding, Gap, Observation) per the classification table |
| 4 | Observe AuditLog assembly | Agent produces AuditLog YAML, validates via `validate_gemara_artifact(#AuditLog)` |

### HP-4: Phase 3 — Cross-framework coverage

| Step | Action | Expected |
|:-----|:-------|:---------|
| 1 | Observe MappingDocument join | Agent joins AuditResults with each MappingDocument's entries |
| 2 | Observe coverage classification | Agent uses `MappingTarget.strength` and `confidence-level` to classify each external framework entry (Covered, Partially Covered, Weakly Covered, Not Covered, Needs Review, Unmapped) |
| 3 | Observe coverage matrix | Agent presents per-framework summary with counts by coverage status |
| 4 | Observe attention items | Agent presents a table of items needing human attention, sorted by risk (Not Covered first) |

### HP-5: Phase 4 — Multi-document output

| Step | Action | Expected |
|:-----|:-------|:---------|
| 1 | Observe artifact emission | Editor receives AuditLog YAML; `detectDefinition()` identifies `#AuditLog` |
| 2 | Inspect YAML | One AuditLog per target separated by `---`; each document has distinct `target.id` |
| 3 | Click Validate | Each AuditLog document passes `validate_gemara_artifact(#AuditLog)` independently |
| 4 | Observe summary | Agent presents cross-framework summary conversationally (not as YAML artifact) |
| 5 | Click Download YAML | YAML file downloaded to local filesystem |

### HP-6: Conversation flow

| Step | Action | Expected |
|:-----|:-------|:---------|
| 1 | Count total exchanges | ~4-6 exchanges: scope confirmation, inventory confirmation, per-target results, cross-framework summary |
| 2 | Verify propose-confirm pattern | Agent proposes defaults (tables, summaries), waits for user confirmation at phase boundaries |
| 3 | Verify no open-ended questions | Agent derives values from data, does not ask "what targets do you want?" without first presenting discovered inventory |

---

## Edge Cases

### EC-1: Missing required inputs

| Case | Action | Expected |
|:-----|:-------|:---------|
| No Policy | Start job with MappingDocuments + timeline but no Policy | Agent responds: "I need a Policy and audit timeline (start/end dates) to prepare the audit. Please provide them." |
| No timeline | Provide Policy + MappingDocuments but no dates | Same guidance message requesting both Policy and timeline |
| No MappingDocuments | Provide Policy + timeline only | Agent responds: "I can assess evidence against your criteria, but without MappingDocuments I can't translate coverage to external frameworks. Want to proceed with internal-only analysis, or import MappingDocuments first?" |

### EC-2: Empty evidence

| Case | Action | Expected |
|:-----|:-------|:---------|
| No targets found | Provide a policy_id with zero evidence rows in ClickHouse for the given timeline | Agent reports no targets discovered, does not fabricate inventory |
| Target with zero evaluations | Confirm a target that has enforcement data but no evaluation data | All criteria entries classified as Gap for that target |
| policy_id mismatch | Provide a policy_id that doesn't match any ClickHouse rows | Agent reports empty inventory, suggests verifying the policy_id |

### EC-3: Cadence validation

| Case | Action | Expected |
|:-----|:-------|:---------|
| Complete cadence | Policy says daily, evidence has daily entries for full window | No cadence Findings; criteria classified purely by result/disposition |
| Missing cycles | Policy says daily, 90-day window, evidence missing 3 days | Agent produces Findings for each missing day with specific dates (e.g., "No evaluation on Feb 12, Mar 3, Mar 17") |
| Weekly frequency | Policy says weekly, 12-week window, evidence missing 2 weeks | Agent reports 2 cadence Findings with the specific week ranges |
| Mixed frequencies | Policy has some plans at daily, others at monthly | Each plan's cadence validated independently against its own frequency |

### EC-4: MappingDocument quality

| Case | Action | Expected |
|:-----|:-------|:---------|
| Missing strength scores | MappingDocument targets have no `strength` field | Agent warns user about missing strength fields; defaults to Undetermined coverage |
| Missing confidence-level | MappingDocument targets have strength but no `confidence-level` | Agent warns about missing confidence; classification proceeds with strength only |
| Invalid MappingDocument | Malformed YAML or fails `validate_gemara_artifact(#MappingDocument)` | Agent reports validation error, requests corrected MappingDocument |
| MappingDocument with `no-match` relationships | Some mappings have `relationship: no-match` (no targets) | Those source entries classified as Unmapped in the framework coverage |

### EC-5: Cross-framework coverage edge cases

| Case | Action | Expected |
|:-----|:-------|:---------|
| Multiple controls → same external entry | Two internal controls map to SOC 2 CC6.1 with strength 8 and strength 4 | External entry classified using strongest (8 → Covered); weaker mapping noted in recommendations |
| External entry with no source mapping | MappingDocument target entry not referenced by any mapping source | Entry appears as Unmapped in the coverage matrix |
| All entries covered | Every external framework entry maps to a Strength result with strength 8+ | Coverage matrix shows 100% Covered; no attention items |
| All entries gaps | No evidence for any criteria | Every external entry shows Not Covered; attention items list is exhaustive |

### EC-6: Multi-target edge cases

| Case | Action | Expected |
|:-----|:-------|:---------|
| Single target only | ClickHouse returns only one distinct target | Agent still presents inventory table (1 row); proceeds normally with single-target AuditLog |
| User filters targets | 5 targets discovered, user selects 2 | Agent produces AuditLogs only for the 2 confirmed targets |
| Target with sparse evidence | One target has full evidence, another has evidence for 50% of criteria | Both targets get complete AuditLogs; sparse target has Gaps for missing criteria |

### EC-7: ClickHouse unavailability

| Case | Action | Expected |
|:-----|:-------|:---------|
| ClickHouse down | clickhouse-mcp returns error on `run_select_query` | Agent reports the issue and halts; does not attempt classification without evidence |
| Query timeout | ClickHouse query exceeds timeout | Agent reports timeout, does not fall back to fabricated data |

### EC-8: Output validation

| Case | Action | Expected |
|:-----|:-------|:---------|
| Multi-doc YAML structure | Inspect emitted YAML | Each document starts with `---`, has its own `metadata.type: AuditLog` and unique `metadata.id` |
| AuditResult completeness | Count AuditResults vs criteria entries | Every criteria entry has exactly one AuditResult per target; no missing, no duplicates |
| Recommendations contain framework context | Inspect `AuditResult.recommendations[]` on a Partially Covered result | At least one recommendation references the external framework entry and mapping strength |
| Cross-framework summary is not YAML | Inspect the summary output | Presented as formatted text in chat, not as a Gemara artifact YAML block |

---

## Helm Verification

| Check | Command | Expected |
|:------|:--------|:---------|
| CRD description | `helm template studio charts/complytime-studio/ \| grep -A2 "name: studio-gap-analyst"` | `spec.description` contains "combined audit preparation" |
| Prompt embedded | Inspect rendered `systemMessage` for gap-analyst | Contains all 4 phases, 12 steps, cadence validation, cross-framework coverage table |
| A2A skill | Inspect rendered `a2aConfig.skills` for gap-analyst | name: "Combined Audit Preparation", tags include "combined-audit" |
| Agent directory | Inspect `AGENT_DIRECTORY` env var in gateway deployment | gap-analyst entry matches updated description and skills |
| No stale refs | `grep -r "single-target\|evidence synthesis from pre-evaluated\|Analyze a MappingDocument to classify" charts/` | Zero matches |
