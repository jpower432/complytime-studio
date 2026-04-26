## Context

The policy drill-down at `#posture/{id}` is the primary analyst surface after the posture overview. It currently has three tabs (Requirements, Evidence, History). Requirements works well — control-to-requirement-to-evidence with classifications, risk badges, and export. Evidence and History are underbuilt. There is no inventory view despite the data existing in the backend.

All data needed for this redesign is already queryable. Evidence rows include `target_id`, `target_name`, `control_id`, `eval_result`, `engine_name`, and `collected_at`. Audit logs include summary JSON with `strengths`, `findings`, `gaps`, `observations`. No new backend work is required.

## Decisions

**1. Add Inventory tab between Requirements and Evidence**
- Targets: GROUP BY `target_id` on evidence already fetched for the policy. Show name, evidence count, and mini posture bar (pass/fail/other).
- Controls: GROUP BY `control_id` on evidence. Show ID, evidence count, pass rate.
- Does NOT include threats or risks. Those read APIs exist for downstream graph DB consumers, not analyst views.
- Alternative: query the `controls` table to show zero-evidence controls as gaps. Rejected for this pass — adds a new endpoint dependency. Evidence-only grouping shows what's actually being assessed.

**2. Evidence tab: summary strip + recency fading, remove dead filter**
- Summary strip computed client-side: total records, pass rate, distinct engine count, distinct target count. Same pattern as `PostureSummary` in posture-view.
- Remove the "framework" filter. `GET /api/evidence` has no framework parameter — the field was always dead UI. Framework is a mapping-level concept, not evidence-level.
- Row-level recency: apply opacity + CSS class based on `collected_at` age. Thresholds match posture card freshness (7d / 30d / 90d). Data stays in the DB — this is visual-only.
- Recency thresholds: extract `freshnessClass` and `STALE_THRESHOLD_DAYS` to a shared utility (currently in `posture-view.tsx`).

**3. History tab: table with deltas instead of cards**
- Cards → table. Columns: Period, Framework, Strengths, Findings, Gaps, Author.
- Delta values (e.g. `+2`, `−1`) computed by diffing `summary` JSON of adjacent rows (ordered by `audit_start DESC`). Positive finding/gap deltas get a warning color. Negative deltas get a success color.
- Audit ID filter: `<select>` dropdown populated from fetched `logs` when embedded. Standalone view keeps the text input for cross-policy lookup.
- Click row to expand inline YAML viewer (same expand pattern as requirement evidence drill-down). Replaces the disconnected "Audit Detail" panel.

**4. Tab order**
- `[Requirements | Inventory | Evidence | History]`
- Requirements stays default since it's the primary analyst workflow.

## Risks / Trade-offs

- **Client-side aggregation** — Inventory and evidence summary are computed from the full evidence response. For policies with thousands of evidence rows, this is fine (the API already caps at `MaxQueryLimit = 1000`). If we later need server-side aggregation, add dedicated summary endpoints.
- **No zero-evidence control detection** — Inventory only shows controls that appear in evidence. Controls that exist in the `controls` table but have no evidence won't appear. Acceptable for this pass; the Requirements tab already surfaces these as "No Evidence" classification.
- **Freshness thresholds hardcoded** — 7d / 30d / 90d are reasonable defaults. Not configurable per-policy yet. Could be in the future if different policies have different assessment cadences.
