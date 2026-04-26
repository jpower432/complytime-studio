## Why

The evidence page lacks filtering depth, staleness visibility, and cross-view navigation. Users cannot drill from inventory into scoped evidence, cannot filter by most evidence fields, and have no visual signal for evidence freshness beyond a per-row text badge. The policy detail evidence tab also exposes upload controls that belong exclusively on the main evidence management page.

## What Changes

- Inventory items (targets, controls) become clickable, navigating to the Evidence tab with a scoped filter chip
- Evidence filter bar gains an "+ Filter" menu for secondary filters (target, result, engine, compliance status, owner, enrichment status) rendered as dismissible chips
- Evidence freshness is frequency-aware, driven by `Policy.adherence.assessment-plans[].frequency` with a 30-day fallback
- Interactive freshness bar replaces static summary counts — click a segment to filter by freshness bucket
- Evidence row backgrounds use neutral-tint gradient (mode-adaptive) to communicate staleness at a glance
- Policy detail evidence tab hides upload/manual entry controls (read-only)
- Audit history delta values (`+1`, `-1`, `0`) gain tooltips explaining the comparison to prior audit

## Capabilities

### New Capabilities
- `filter-chip-system`: Reusable dismissible filter chip component and interaction pattern used across all filterable views
- `evidence-freshness-bar`: Interactive segmented freshness bar that visualizes staleness distribution and acts as a filter entry point
- `frequency-aware-staleness`: Client-side staleness calculation using policy assessment plan frequency with fallback threshold
- `evidence-secondary-filters`: "+ Filter" menu with dynamic field selection for target, result, engine, compliance status, owner, enrichment status

### Modified Capabilities
- `evidence-tab-fix`: Policy-scoped evidence tab hides upload/management controls when embedded
- `inventory-tab`: Inventory items become clickable, setting a scoped filter chip on the evidence tab
- `history-tab-fix`: Audit history delta cells gain tooltip explaining change from prior audit

## Impact

- `workbench/src/components/evidence-view.tsx` — filter bar, freshness bar, row tinting, embedded read-only gate
- `workbench/src/components/inventory-view.tsx` — click handlers on target/control items
- `workbench/src/components/audit-history-view.tsx` — tooltip on DeltaCell
- `workbench/src/lib/freshness.ts` — frequency-aware staleness calculation
- `workbench/src/app.tsx` — new signals for target filter, freshness filter
- CSS — neutral tint palette, chip styles, freshness bar segments
- Decision docs already captured: `filter-chip-pattern.md`, `evidence-staleness-model.md`, `evidence-filter-bar.md`
