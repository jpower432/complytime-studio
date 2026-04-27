## 1. Shared Freshness Utility

- [x] 1.1 Extract `STALE_THRESHOLD_DAYS`, `isStale`, `freshnessClass` from `posture-view.tsx` into `workbench/src/lib/freshness.ts`
- [x] 1.2 Add `evidenceRecencyClass(collectedAt: string)` returning `recency-current` / `recency-aging` / `recency-stale` / `recency-very-stale` with thresholds at 7d / 30d / 90d
- [x] 1.3 Update `posture-view.tsx` to import from shared utility
- [x] 1.4 Add CSS classes for evidence recency (opacity + optional stale badge)

## 2. Inventory Tab

- [x] 2.1 Add `"inventory"` to the `TABS` array in `policy-detail-view.tsx` between Requirements and Evidence
- [x] 2.2 Create `workbench/src/components/inventory-view.tsx` accepting `policyIdOverride`
- [x] 2.3 Fetch evidence via `GET /api/evidence?policy_id={id}&limit=1000`
- [x] 2.4 Client-side GROUP BY `target_id` → target list with name, count, mini posture bar
- [x] 2.5 Client-side GROUP BY `control_id` → control list with ID, count, pass rate
- [x] 2.6 Add CSS for inventory layout (two-column grid, mini posture bars)

## 3. Evidence Tab Fix

- [x] 3.1 Remove `framework` state variable and the "Framework" input from filter bar
- [x] 3.2 Remove the "More Filters" toggle button and the hidden `evidence-filters-extra` section
- [x] 3.3 Add `EvidenceSummary` component computing totals from returned records
- [x] 3.4 Apply `evidenceRecencyClass` to each evidence table row based on `collected_at`
- [x] 3.5 Add stale badge rendering for rows older than 30d

## 4. History Tab Fix

- [x] 4.1 Replace `audit-card` article elements with a `<table>` layout
- [x] 4.2 Add delta computation: diff `parseSummary(logs[i])` vs `parseSummary(logs[i+1])` for adjacent rows
- [x] 4.3 Style positive deltas (finding/gap increase) with warning color, negative with success color
- [x] 4.4 Replace Audit ID `<input>` with `<select>` when `embedded === true`, populated from `logs`
- [x] 4.5 Add click-to-expand on table rows with inline YAML `<pre>` viewer + Download YAML button
- [x] 4.6 Remove the separate `audit-detail` panel and its state management

## 5. Tab Bar Update

- [x] 5.1 Update `activeTab` signal type to include `"inventory"`
- [x] 5.2 Update `policy-detail-view.tsx` TABS constant and tab content rendering
- [x] 5.3 Update hash routing to support `tab=inventory`
