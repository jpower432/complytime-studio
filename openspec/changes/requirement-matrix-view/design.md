## Context

Compliance analysts need a bookmarkable, glanceable requirement-level grid (framework → control → requirement → evidence and posture) that uses the same ClickHouse facts as the Studio Assistant, per [cloud-native posture correction](../../docs/decisions/cloud-native-posture-correction.md). PostureView today surfaces policy-level posture; the matrix closes the gap to requirement-level readiness without chat as the only path. The Gateway already serves the Preact workbench, persists `evidence_assessments`, and (per existing pipeline design) defines or will define `unified_compliance_state` and `policy_posture` as query-time views over `evidence` and `evidence_assessments`.

## Goals / Non-Goals

**Goals**

- REST APIs for matrix listing and per-requirement evidence drill-down, scoped by `policy_id` and audit window.
- Workbench route with filters aligned to analyst workflows: policy, control family, classification, staleness, audit window.
- Navigation from PostureView into the matrix with policy (and time) context preserved.
- No new ClickHouse tables for v1; reuse `assessment_requirements`, `evidence`, `evidence_assessments`, `controls`, and existing posture views where they reduce duplication.

**Non-Goals**

- Changing the agent, A2A protocol, or assessment artifact formats.
- Authoring or editing policies or catalogs in the workbench.
- Export formats (CSV/Excel/PDF) for the matrix in this change—tracked as a follow-up unless explicitly pulled in.
- Graph databases or a second query engine.

## Decisions

### Query strategy: ClickHouse views vs gateway-side JOINs

**Decision:** Implement requirement-matrix-specific queries in the Gateway (SQL strings in `internal/store` or a dedicated query module) that **compose** the same join logic as the existing `unified_compliance_state` pattern: `evidence` joined to `evidence_assessments FINAL` for latest classification, plus joins to `assessment_requirements` and `controls` (and policy-scoped resolution via `catalogs` / `policy_id` columns as they exist in schema). If `unified_compliance_state` is available and its columns are sufficient, **prefer** selecting from that view (or a thin wrapper `VIEW` in ClickHouse) for the evidence side of the join to keep semantics aligned with `policy_posture` summaries. If the requirement listing needs columns not in `unified_compliance_state`, **extend** the gateway query with additional JOINs to base tables rather than forking two incompatible JOIN definitions long-term; optionally add a new ClickHouse `VIEW` in a follow-up migration so the agent, REST, and ad-hoc SQL share one definition.

**Rationale:** Pipeline documentation already states query-time views stay current when assessments arrive independently of evidence inserts. Centralizing the evidence+assessment shape avoids drift with `policy_posture` aggregates.

### Pagination for large requirement sets

**Decision:** The list endpoint **SHALL** use cursor- or offset-based pagination with a documented default `limit` (e.g. 50–100) and a maximum cap. **SHOULD** use `limit` + `offset` or `page` for v1 if total counts are needed for the UI; **SHOULD** move to keyset (cursor) pagination if sort order is stable and performance requires it at scale. Require `policy_id` always so scans stay partition-friendly.

**Rationale:** A single large JSON payload would hurt TTFB and workbench memory; analysts still need to scan full frameworks eventually via pages or "load more."

### How `evidence_assessments` classification joins

**Decision:** For each `evidence_id`, use `evidence_assessments FINAL` (or `argMax` over `assessed_at`) to pick the **latest** classification. Roll up to the requirement row with explicit rules: e.g. worst-of across evidence rows, "any unassessed", or primary evidence—**document the rule in the API** so the matrix is deterministic. Staleness: derive from `collected_at` vs policy thresholds and/or `classification = 'Stale'` from assessments, matching how `policy_posture` surfaces staleness.

**Rationale:** `ReplacingMergeTree` / history semantics require a single "current" view for REST consumers.

### Workbench component architecture

**Decision:** Add a new hash route (e.g. `#/requirements` or equivalent pattern used by PostureView) and a sidebar entry. Data fetching: use the existing `apiFetch` pattern and shared app signals (`selectedPolicyId`, `selectedTimeRange`) from `app.ts` for consistency with `chat-assistant` context injection. Matrix state: local component state for filters; optional URL query sync for shareable links. New Preact component(s) under `workbench/src/components/` with a thin API module (e.g. `api/requirements.ts`) for types and request builders.

**Rationale:** Matches established workbench patterns, keeps the agent context block aligned with the visible policy and time range.

### Relationship to `policy_posture` and `unified_compliance_state`

**Decision:** `policy_posture` remains the **aggregated** per-policy (and target) summary for PostureView and high-level health. `unified_compliance_state` remains the **row-level** join of evidence plus latest assessment for drill-down. The requirement matrix is **requirement-grained**: aggregate evidence and assessments *per `requirement_id`* (and control/catalog), potentially grouping evidence through `evidence.requirement_id` and `assessment_requirements` keys. The matrix APIs **MUST** use classification and time semantics compatible with `unified_compliance_state` so a user who drills from a policy card into the matrix does not see contradictory pass/fail narratives versus PostureView.

**Rationale:** One mental model: PostureView = policy aggregate; matrix = requirement decomposition of the same underlying facts.

## Risks / Trade-offs

| Risk / trade-off | Mitigation |
|:--|:--|
| JOIN cost across large `evidence` × requirements | Mandatory `policy_id` + time filters; indexes/order keys as already defined; pagination; consider pre-aggregated ClickHouse `VIEW` later. |
| Semantic mismatch between matrix roll-up and `policy_posture` | Document roll-up rules; add integration tests with seeded data; consider reusing a single subquery from `unified_compliance_state`. |
| `assessment_requirements` not keyed by `policy_id` in the same way as `evidence` | Join via `catalog_id` and policy import linkage (`controls.policy_id` / `catalogs.policy_id`); test policies with multiple catalogs. |
| Viewer vs admin: matrix is read-only | GET-only endpoints align with existing role model; no new write paths. |
| Staleness definition ambiguity | Expose both rule-based date staleness and enum `Stale` from assessments where both apply; document precedence in the OpenAPI or spec. |
