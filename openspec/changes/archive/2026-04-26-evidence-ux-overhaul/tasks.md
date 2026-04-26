## 1. Filter Chip System

- [x] 1.1 Create `FilterChip` component in `workbench/src/components/filter-chip.tsx` — renders `"Label: value ✕"` with dismiss callback
- [x] 1.2 Create `useFilterChips` hook managing a `Map<string, string>` of active filters with add/remove/clear helpers
- [x] 1.3 Add CSS for filter chips (`.filter-chip`, `.filter-chip-dismiss`) with light/dark mode support
- [x] 1.4 Render active filter chips above the evidence data table in `evidence-view.tsx`

## 2. Frequency-Aware Staleness

- [x] 2.1 Add `FREQUENCY_TO_DAYS` map and `frequencyToDays()` function in `freshness.ts` (daily=1, weekly=7, monthly=30, quarterly=90, annually=365, on-demand=Infinity)
- [x] 2.2 Add `parsePolicyFrequencies(contentYaml: string): Map<string, number>` to extract `requirement_id → cycle_days` from policy YAML `adherence.assessment-plans[]`
- [x] 2.3 Add `freshnessFromFrequency(collectedAt: string, cycleDays: number): FreshnessBucket` returning Current/Aging/Stale/VeryStale based on age-to-cycle ratio
- [x] 2.4 Update `EvidenceView` to parse policy content and apply frequency-aware freshness per row when `policyIdOverride` is set, falling back to 30-day default otherwise

## 3. Evidence Freshness Bar

- [x] 3.1 Create `FreshnessBar` component — segmented bar with proportional widths per bucket, no counts or percentages
- [x] 3.2 Add neutral mode-adaptive CSS custom properties for freshness segments (`--freshness-current` through `--freshness-very-stale`)
- [x] 3.3 Add tooltip on hover for each segment showing bucket name
- [x] 3.4 Add `onClick` per segment that creates a `Freshness: <bucket>` filter chip via `useFilterChips`
- [x] 3.5 Render `FreshnessBar` above the evidence table in place of static summary counts

## 4. Evidence Row Tinting

- [x] 4.1 Define CSS custom properties for row background tints (light/dark mode) using neutral HSL shades at ~8% opacity
- [x] 4.2 Apply freshness bucket as a CSS class on each `<tr>` in the evidence table
- [x] 4.3 Remove the "stale" text badge — background tint replaces it
- [x] 4.4 Verify text readability against tinted backgrounds in both light and dark mode

## 5. Evidence Secondary Filters

- [x] 5.1 Create `AddFilterMenu` component — button toggles dropdown listing available fields, excludes already-active filters
- [x] 5.2 Implement inline value selector for enum fields (Result: Passed/Failed/Unknown; Compliance Status: Compliant/Non-Compliant/Exempt/Not Applicable/Unknown)
- [x] 5.3 Implement inline value selector for dynamic fields (Target, Engine, Owner, Enrichment Status) populated from current evidence data
- [x] 5.4 Wire value selection to create a filter chip and trigger client-side re-filter
- [x] 5.5 Integrate `AddFilterMenu` into the evidence filter bar between the date inputs and Search button

## 6. Inventory Click-Through

- [x] 6.1 Add `onClick` handler to target items in `inventory-view.tsx` that calls a callback with `target_id`
- [x] 6.2 Add `onClick` handler to control items in `inventory-view.tsx` that calls a callback with `control_id`
- [x] 6.3 Wire click handlers in `policy-detail-view.tsx` to switch to evidence tab and set the corresponding filter chip
- [x] 6.4 Add cursor pointer and hover state to inventory items

## 7. Embedded Read-Only Gate

- [x] 7.1 Change upload button condition in `evidence-view.tsx` from `role === "admin"` to `!embedded && role === "admin"`
- [x] 7.2 Verify upload button and manual entry form are hidden when evidence tab is embedded in policy detail

## 8. Audit History Delta Tooltips

- [x] 8.1 Add `title` attribute to delta `<span>` in `DeltaCell` — positive: "N more than prior audit", negative: "N fewer than prior audit", zero: "No change from prior audit"
- [x] 8.2 Verify tooltips render on hover in both light and dark mode
