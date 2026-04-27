## Why

The posture drill-down view (`#posture/{id}`) has three tabs — Requirements, Evidence, History. Requirements is solid. Evidence and History are not. Evidence shows a flat table with a dead "framework" filter, no summary, no inventory breakdown, and no visual signal for stale rows. History renders audit logs as disconnected cards with a raw text Audit ID input — no comparison, no timeline, no way to see trend direction. There is no inventory view at all — the posture card shows target and control counts but the drill-down has nowhere to see *which* targets and controls are in scope.

## What Changes

Frontend-only redesign of the policy drill-down. Zero new backend endpoints. All aggregation is client-side from existing API responses.

### Capabilities

#### New Capabilities

- `inventory-tab`: New tab showing in-scope targets and controls with per-item pass/fail posture bars, grouped from existing evidence data

#### Modified Capabilities

- `evidence-tab-fix`: Remove dead framework filter, add summary strip (total records, pass rate, engine count, target count), add row-level recency fading using existing freshness thresholds (7d / 30d / 90d)
- `history-tab-fix`: Replace audit log cards with a table showing period-over-period deltas, replace Audit ID text input with select dropdown when embedded, click-to-expand for YAML detail

## Impact

- Tab bar changes from `[Requirements | Evidence | History]` to `[Requirements | Inventory | Evidence | History]`
- Evidence tab loses the broken "framework" filter input (and the hidden "target name" filter behind "More Filters")
- History tab layout changes from card grid to table with inline expand
- No backend changes. No new API endpoints. No store modifications.
- Freshness thresholds reuse existing constants from `posture-view.tsx` (7d current, 30d aging, >30d stale)
