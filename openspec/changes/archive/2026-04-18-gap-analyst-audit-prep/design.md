## Context

The Gap Analyst specialist synthesizes pre-evaluated L5/L6 evidence from ClickHouse against L3 Policy criteria to produce L7 AuditLog artifacts. The evidence-clickhouse change established the infrastructure. This change evolves the agent from a single-target evidence classifier into a combined audit preparation assistant.

**Persona**: Compliance analyst preparing for a combined audit (e.g., SOC 2 + ISO 27001 + FedRAMP simultaneously). They own the internal Policy and need to validate coverage across multiple external frameworks before the auditor arrives.

**Problem**: Combined audits require cross-framework coverage analysis. A single AuditLog against one target tells you what passed or failed — it doesn't tell you whether your SOC 2 CC6.1 obligation is satisfied when your internal control only partially maps to it (strength: 6, confidence: Medium). The analyst needs that translation layer, surfaced factually, so they can prioritize remediation.

## Goals / Non-Goals

**Goals:**

- Derive audit target inventory from distinct `target_id` values in ClickHouse evidence
- Accept user-provided audit timeline (start/end) instead of deriving from policy frequency
- Use MappingDocuments to translate internal criteria coverage into external framework coverage
- Surface partial coverage using `MappingTarget.strength` and `MappingTarget.confidence-level` scores
- Produce one AuditLog per target in a multi-YAML-document file with a cross-framework summary

**Non-Goals:**

- Authoring MappingDocuments (users bring their own or import from registry)
- Defining pass/fail thresholds (the agent surfaces factual coverage data; humans decide what's acceptable)
- Real-time evidence streaming (batch evidence already in ClickHouse)
- Replacing the auditor's judgment (this is prep, not certification)

## Data Flow

### End-to-End: Workbench → Agent → Output

```
┌─ WORKBENCH ─────────────────────────────────────────────────────────────┐
│                                                                         │
│  User selects "studio-gap-analyst"                                      │
│  User provides:                                                         │
│    1. Policy (YAML or policy_id reference)                              │
│    2. MappingDocuments (1..N, for each external framework)              │
│    3. Audit timeline (start date, end date)                             │
│                                                                         │
│       ▼  POST /api/a2a/studio-gap-analyst                               │
│     message/send { role: "user", parts: [...] }                         │
│                                                                         │
│  streamTask(taskId) ──▶ SSE stream                                      │
│                                                                         │
│  ◄── GUIDED CONVERSATION (4-6 exchanges) ──▶                            │
│  │                                                                      │
│  │  Phase 1: Scope & Inventory                                          │
│  │  ├─ Agent confirms policy, timeline, frameworks                      │
│  │  ├─ Agent queries ClickHouse for distinct targets                    │
│  │  ├─ Agent presents target inventory table                            │
│  │  └─ User confirms/filters targets                                    │
│  │                                                                      │
│  │  Phase 2: Evidence Assessment                                        │
│  │  ├─ Agent queries evidence per target within timeline                │
│  │  ├─ Agent classifies criteria (Gap/Finding/Observation/Strength)     │
│  │  └─ Agent presents per-target summary                                │
│  │                                                                      │
│  │  Phase 3: Cross-Framework Coverage                                   │
│  │  ├─ Agent joins AuditResults with MappingDocument entries            │
│  │  ├─ Agent uses strength + confidence to classify coverage            │
│  │  ├─ Agent presents cross-framework coverage matrix                   │
│  │  └─ User reviews, asks about specific gaps                           │
│  │                                                                      │
│  │  Phase 4: Output                                                     │
│  │  ├─ Agent emits multi-YAML-doc (one AuditLog per target)             │
│  │  └─ Agent emits cross-framework summary                              │
│  │                                                                      │
│  ├─ onArtifact ──▶ detectDefinition() ──▶ #AuditLog                     │
│  │   └──▶ setEditorArtifact() ──▶ editor updates                        │
│  └─ onDone("completed")                                                 │
│                                                                         │
│  Post-authoring: Validate / Save / Publish                              │
└─────────────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─ GATEWAY ───────────────────────────────────────────────────────────────┐
│  /api/a2a/studio-gap-analyst                                            │
│  ├─ auth.Middleware extracts GitHub token from session cookie            │
│  └─ ReverseProxy ──▶ http://studio-gap-analyst:8080/invoke              │
│      └─ injects Authorization: Bearer {github_token}                    │
└──────────────────────────────┬──────────────────────────────────────────┘
                               ▼
┌─ AGENT POD (kagent) ────────────────────────────────────────────────────┐
│  platform.md + prompt.md ──▶ system prompt                              │
│  skills: gemara-layers, audit-classification (git clone)                │
│  model: claude-sonnet-4                                                 │
│                                                                         │
│  MCP CONNECTIONS:                                                       │
│  ┌─────────────────────────┐                                            │
│  │ gemara-mcp (stdio)      │ ◄── validate_gemara_artifact              │
│  │                         │     migrate_gemara_artifact                │
│  ├─────────────────────────┤                                            │
│  │ github-mcp (http+OBO)   │ ◄── get_file_contents                    │
│  │                         │     search_code                           │
│  ├─────────────────────────┤                                            │
│  │ clickhouse-mcp (stdio)  │ ◄── run_select_query                     │
│  │                         │     list_tables                           │
│  │                         │     list_databases                        │
│  └─────────────────────────┘                                            │
└─────────────────────────────────────────────────────────────────────────┘
```

### Phase 1: Inventory Derivation

The agent discovers targets rather than requiring them upfront.

```
┌─────────────────────────────────────────────────────────────────────┐
│  USER INPUT                                                         │
│  ├─ policy_id: "pol-cloud-security-v2"                              │
│  ├─ audit_start: "2026-03-01"                                       │
│  ├─ audit_end: "2026-04-18"                                         │
│  └─ frameworks: [SOC2-mapping.yaml, ISO27001-mapping.yaml]          │
└────────────────────┬────────────────────────────────────────────────┘
                     ▼
┌─────────────────────────────────────────────────────────────────────┐
│  CLICKHOUSE QUERY                                                   │
│                                                                     │
│  SELECT DISTINCT target_id, target_name, target_type,               │
│         min(collected_at) AS earliest, max(collected_at) AS latest,  │
│         count(*) AS eval_count                                      │
│  FROM evaluation_logs                                               │
│  WHERE policy_id = 'pol-cloud-security-v2'                          │
│    AND collected_at BETWEEN '2026-03-01' AND '2026-04-18'           │
│  GROUP BY target_id, target_name, target_type                       │
│  ORDER BY target_id                                                 │
└────────────────────┬────────────────────────────────────────────────┘
                     ▼
┌─────────────────────────────────────────────────────────────────────┐
│  INVENTORY TABLE (presented to user)                                │
│                                                                     │
│  | Target              | Type     | Evals | Window          |       │
│  |:--------------------|:---------|------:|:----------------|       │
│  | prod-cluster-east   | Software | 312   | Mar 1 – Apr 18  |       │
│  | prod-cluster-west   | Software | 298   | Mar 3 – Apr 18  |       │
│  | staging-cluster     | Software | 145   | Mar 1 – Apr 15  |       │
│                                                                     │
│  User confirms: "Audit prod-cluster-east and prod-cluster-west"     │
└─────────────────────────────────────────────────────────────────────┘
```

### Phase 2: Evidence Assessment (per target)

For each confirmed target, the agent queries and classifies.

```
┌─ PER TARGET ────────────────────────────────────────────────────────┐
│                                                                     │
│  1. Load criteria from Policy                                       │
│     └─ imports.catalogs[] ──▶ extract controls + requirements       │
│        = complete criteria set                                      │
│                                                                     │
│  2. Query evaluation evidence                                       │
│     └─ SELECT * FROM evaluation_logs                                │
│        WHERE policy_id = ? AND target_id = ?                        │
│          AND collected_at BETWEEN ? AND ?                           │
│        ORDER BY control_id, requirement_id, collected_at DESC       │
│                                                                     │
│  3. Query enforcement evidence                                      │
│     └─ SELECT * FROM enforcement_actions                            │
│        WHERE policy_id = ? AND target_id = ?                        │
│          AND started_at BETWEEN ? AND ?                             │
│                                                                     │
│  4. Classify each criteria entry                                    │
│     ┌────────────────────────────────────────────────────────────┐  │
│     │  L5 Result    │ L6 Disposition │ AuditResult Type          │  │
│     │───────────────│────────────────│───────────────────────────│  │
│     │  Passed       │ Clear          │ Strength                  │  │
│     │  Passed       │ Tolerated      │ Observation               │  │
│     │  Failed       │ Enforced       │ Finding (remediated)      │  │
│     │  Failed       │ Tolerated      │ Finding (accepted risk)   │  │
│     │  Failed       │ none           │ Finding                   │  │
│     │  (no data)    │ (no data)      │ Gap                       │  │
│     │  Needs Review │ any            │ Observation               │  │
│     └────────────────────────────────────────────────────────────┘  │
│                                                                     │
│  5. Assemble AuditLog for this target                               │
│     └─ One AuditResult per criteria entry (completeness required)   │
│     └─ validate_gemara_artifact(#AuditLog)                          │
└─────────────────────────────────────────────────────────────────────┘
```

### Phase 3: Cross-Framework Coverage

This is the new capability. MappingDocuments bridge internal criteria to external frameworks.

```
┌─ CROSS-FRAMEWORK JOIN ──────────────────────────────────────────────┐
│                                                                     │
│  For each MappingDocument:                                          │
│                                                                     │
│  MappingDocument.source-reference ──▶ Internal Policy/Catalog       │
│  MappingDocument.target-reference ──▶ External Framework            │
│  MappingDocument.mappings[] ──▶ source entry → target entries       │
│                                                                     │
│  JOIN LOGIC:                                                        │
│                                                                     │
│  ┌─────────────────┐     ┌──────────────────┐     ┌──────────────┐ │
│  │  AuditResults    │     │  MappingDocument  │     │  External    │ │
│  │  (per target)    │     │  .mappings[]      │     │  Framework   │ │
│  │                  │     │                   │     │  Entries     │ │
│  │  result.criteria │────▶│  mapping.source   │     │              │ │
│  │  -reference      │     │       │           │     │              │ │
│  │                  │     │       ▼           │     │              │ │
│  │                  │     │  mapping.targets[]│────▶│  CC6.1       │ │
│  │                  │     │    .entry-id      │     │  CC6.2       │ │
│  │                  │     │    .strength (1-10│)    │  CC7.1       │ │
│  │                  │     │    .confidence-   │     │  ...         │ │
│  │                  │     │     level         │     │              │ │
│  └─────────────────┘     └──────────────────┘     └──────────────┘ │
│                                                                     │
│  COVERAGE CLASSIFICATION (per external framework entry):            │
│                                                                     │
│  ┌────────────────────────────────────────────────────────────────┐ │
│  │ AuditResult  │ Strength │ Confidence │ Framework Coverage      │ │
│  │──────────────│──────────│────────────│────────────────────────│ │
│  │ Strength     │ 8-10     │ High       │ Covered                │ │
│  │ Strength     │ 5-7      │ Med/High   │ Partially Covered      │ │
│  │ Strength     │ 1-4      │ any        │ Weakly Covered         │ │
│  │ Finding      │ any      │ any        │ Not Covered (finding)  │ │
│  │ Gap          │ any      │ any        │ Not Covered (no data)  │ │
│  │ Observation  │ any      │ any        │ Needs Review           │ │
│  │ (no mapping) │ —        │ —          │ Unmapped               │ │
│  └────────────────────────────────────────────────────────────────┘ │
│                                                                     │
│  Multiple internal controls may map to one external entry.          │
│  Use the strongest coverage when multiple mappings exist.           │
└─────────────────────────────────────────────────────────────────────┘
```

### Cross-Framework Summary (presented to user)

```
┌─ COVERAGE MATRIX ───────────────────────────────────────────────────┐
│                                                                     │
│  SOC 2 Trust Services Criteria (73 entries)                         │
│  ├─ Covered:           51 (69.9%)                                   │
│  ├─ Partially Covered: 12 (16.4%)  ◄── strength 5-7, actionable    │
│  ├─ Weakly Covered:     4 (5.5%)   ◄── strength 1-4, high risk     │
│  ├─ Not Covered:         3 (4.1%)  ◄── findings or gaps            │
│  ├─ Needs Review:        1 (1.4%)                                   │
│  └─ Unmapped:            2 (2.7%)                                   │
│                                                                     │
│  ISO 27001 Annex A (93 entries)                                     │
│  ├─ Covered:           68 (73.1%)                                   │
│  ├─ Partially Covered:  9 (9.7%)                                    │
│  ├─ Weakly Covered:     7 (7.5%)                                    │
│  ├─ Not Covered:         5 (5.4%)                                   │
│  ├─ Needs Review:        2 (2.2%)                                   │
│  └─ Unmapped:            2 (2.2%)                                   │
│                                                                     │
│  ATTENTION ITEMS (sorted by risk):                                  │
│  | Framework | Entry   | Status    | Strength | Gap Detail         │
│  |:----------|:--------|:----------|:---------|:───────────────────│
│  | SOC 2     | CC6.3   | Not Cov.  | —        | No evidence: ...   │
│  | ISO 27001 | A.8.2   | Weak      | 3        | ctrl-enc-01 ...    │
│  | SOC 2     | CC7.2   | Partial   | 5        | ctrl-mon-02 ...    │
│  | ...       |         |           |          |                    │
└─────────────────────────────────────────────────────────────────────┘
```

### Output Structure

```yaml
# --- Document 1: AuditLog for prod-cluster-east ---
metadata:
  type: AuditLog
  id: audit-prod-east-2026Q1
  ...
target:
  id: prod-cluster-east
  ...
criteria:
  - reference-id: pol-cloud-security-v2
    ...
results:
  - id: ar-001
    type: Strength
    ...
---
# --- Document 2: AuditLog for prod-cluster-west ---
metadata:
  type: AuditLog
  id: audit-prod-west-2026Q1
  ...
target:
  id: prod-cluster-west
  ...
results:
  - id: ar-001
    type: Finding
    ...
```

The cross-framework coverage summary is presented conversationally (not a Gemara artifact type). The AuditLog `recommendations` field on each result carries the framework-specific context derived from MappingDocuments.

## Decisions

### D1: Inventory derived from evidence, not user-specified

**Decision**: Query `SELECT DISTINCT target_id` from ClickHouse within the audit window. Present the inventory for user confirmation rather than requiring targets upfront.

**Rationale**: The evidence store already knows what's been measured. Asking the user to enumerate targets duplicates knowledge and risks missing targets or including ones with no evidence.

### D2: User-provided audit timeline replaces policy-frequency derivation

**Decision**: The user provides explicit start and end dates for the audit period. The agent no longer derives the time window from `Policy.adherence.assessment-plans[].frequency`.

**Rationale**: Audit periods are set by the audit engagement, not by policy frequency. A quarterly assessment frequency doesn't mean the audit window is 90 days — it means assessments should occur quarterly. The audit might cover 6 months. Policy frequency remains relevant for identifying assessment cadence gaps within the window.

**Note**: The previous evidence-clickhouse design (D7) derived windows from policy frequency. This supersedes that decision for the audit use case. Policy frequency is still used to flag "expected N assessments in window, found M" as an Observation.

### D3: MappingDocument strength and confidence drive framework coverage classification

**Decision**: Use `MappingTarget.strength` (1-10) and `MappingTarget.confidence-level` (Undetermined/Low/Medium/High) from the Gemara schema to classify how well internal criteria coverage translates to external framework requirements. Do not define pass/fail thresholds.

**Rationale**: External frameworks define minimums that organizations build on. A strength score of 5 with Medium confidence means the mapping author assessed partial coverage — that's a fact the compliance analyst needs to act on. Defining a rollout threshold (e.g., "90% must be Covered") is the organization's decision, not the agent's. The agent surfaces the data; the human decides what's acceptable.

### D4: One AuditLog per target, multi-YAML-document output

**Decision**: Produce one valid `#AuditLog` per target, concatenated with YAML document separators (`---`). Each AuditLog is independently valid against the Gemara schema.

**Rationale**: The `#AuditLog` schema requires a single `target: #Resource`. Multiple targets require multiple documents. Multi-YAML-doc is a standard pattern and keeps each document independently validatable.

### D5: Cross-framework summary is conversational, not a new artifact type

**Decision**: The coverage matrix and attention items are presented as formatted text in the chat, not as a new Gemara artifact. Framework-specific recommendations are embedded in `AuditResult.recommendations[]`.

**Rationale**: No Gemara schema exists for cross-framework coverage summaries. Inventing one is out of scope. The actionable data (per-result recommendations with framework context) lives in the AuditLog where it belongs. The summary table is a presentation concern for the compliance analyst during prep.

### D6: Strongest coverage wins when multiple mappings exist

**Decision**: When multiple internal controls map to the same external framework entry, use the strongest coverage classification (highest strength score with highest confidence).

**Rationale**: If `ctrl-ac-01` maps to SOC 2 CC6.1 with strength 8 and `ctrl-ac-02` also maps with strength 4, the external entry is covered by the stronger mapping. The weaker mapping is still noted in recommendations for context.

### D7: Assessment cadence gaps are Findings, not Observations

**Decision**: The agent computes expected assessment cycles from `Policy.adherence.assessment-plans[].frequency` across the user-provided audit window. Missing cycles are classified as Findings — the target is non-compliant with the policy's assessment cadence.

**Rationale**: Continuous compliance is a policy requirement. If the policy says "daily" and evidence has a 3-day gap, that's not an informational note — it's a compliance failure. Auditors evaluate whether the control was operating effectively throughout the period, not just at a point in time. Missing cadence = missing evidence = non-compliant.

**Example**: Audit window Jan 1 – Mar 31, policy frequency "daily" for `ctrl-ac-01`:
- Expected: ~90 evaluation cycles
- Found: 87 (gaps on Feb 12, Mar 3, Mar 17)
- Result: Finding per missing day, with evidence timestamps documenting the gaps

## Risks / Trade-offs

| Risk | Mitigation |
|:-----|:-----------|
| MappingDocument quality determines framework coverage accuracy | Agent validates MappingDocuments via gemara-mcp before analysis. Warns user if strength/confidence fields are missing (defaults to Undetermined). |
| Large multi-target audits may exceed context | Agent processes targets sequentially. Each target's ClickHouse query is independent. Summary is computed incrementally. |
| Multiple MappingDocuments multiply the join complexity | Each MappingDocument is processed independently against the same AuditResults. No cross-MappingDocument joins needed. |
| Users may expect the agent to define thresholds | Prompt explicitly states the agent surfaces facts, not judgments. Coverage classification is descriptive, not prescriptive. |

## Resolved Questions

- **Cross-framework summary as Gemara artifact type?** No. The summary is a view, not source data. Gemara captures source artifacts. If a formal view is needed, use `go-gemara` SDK rendering (e.g., OSCAL conversion). The AuditLogs are the source; the coverage matrix is a presentation concern.
- **MappingDocument sourcing via agent?** No — start with user-provided MappingDocuments. The agent stays focused on analysis, not sourcing. The workbench Import Dialog already handles OCI registry browsing. Future: embed a comprehensive mapping (e.g., Secure Controls Framework) as a skill/resource the agent always has access to, giving cross-framework coverage "for free."
- **Policy frequency as cadence validation?** Yes — this is mandatory. If the policy says daily assessments and evidence has a gap, that is non-compliance with the policy, not just an observation. The agent computes expected assessment cycles from `Policy.adherence.assessment-plans[].frequency` across the audit window and flags missing cycles as Findings.

## Open Questions

- **Comprehensive mapping resource**: When SCF or equivalent MappingDocuments are available as a skill/resource, the agent could always offer cross-framework analysis without user-provided MappingDocuments. Separate change when the content is ready.
