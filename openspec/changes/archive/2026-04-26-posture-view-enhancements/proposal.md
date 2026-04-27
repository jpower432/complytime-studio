## Why

The posture view displays compliance status as text-only cards (numeric counts, plain timestamps). Analysts must read every card to assess overall health, cannot visually distinguish fresh evidence from stale, and have no cross-policy summary. Time-filtered posture requires navigating to the requirement matrix. These gaps slow triage and reduce the dashboard's value as a quick-scan surface.

## What Changes

- Add a stacked pass/fail/other progress bar to each posture card for instant ratio scanning.
- Add evidence recency coloring (card border) so analysts see freshness without reading timestamps.
- Add an aggregate summary strip above the card grid showing cross-policy pass rate, total counts, and stale-evidence warnings.
- Add time-preset filters (7d / 30d / 90d / All) to the posture view, backed by a new time-range parameter on the posture API.

## Capabilities

### New Capabilities

- `posture-visual-density`: Stacked progress bar, recency coloring, and aggregate summary strip on posture cards.
- `posture-time-filter`: Time-preset buttons on posture view and backend support for time-range filtering on the posture API.

### Modified Capabilities

- `posture-drilldown`: The posture card layout changes (bar + border color added). Drilldown behavior is unchanged, but card markup structure is modified.

## Impact

- **Backend**: `ListPosture` store method and `GET /api/posture` handler gain optional `start`/`end` query parameters. ClickHouse query adds conditional `WHERE` clause on `collected_at`.
- **Frontend**: `posture-view.tsx` gains three new inline components (`PostureBar`, `PostureSummary`, `TimePresets`) and a `freshnessClass` utility. No new dependencies.
- **CSS**: New classes in `global.css` for bar segments, recency border colors, summary strip, and time presets.
- **Tests**: Backend posture handler tests updated for parameterized time filtering. Frontend component tests for new visual elements.
