## 1. Backend: Time-filtered posture API

- [x] 1.1 Update `PostureStore` interface: `ListPosture(ctx context.Context, start, end time.Time) ([]PostureRow, error)`
- [x] 1.2 Update `Store.ListPosture` ClickHouse query to add conditional `WHERE e.collected_at >= ? AND e.collected_at <= ?` when start/end are non-zero
- [x] 1.3 Update `listPostureHandler` to parse optional `start` and `end` query params (date-only or RFC 3339), return 400 on invalid format
- [x] 1.4 Update `QueryPolicyPosture` signature to match (or confirm it does not need time params)
- [x] 1.5 Update `posture_handler_test.go`: add test cases for filtered request, unfiltered request, and invalid date params

## 2. Frontend: Stacked progress bar

- [x] 2.1 Add `PostureBar` component to `posture-view.tsx` (CSS flex, percentage widths, `role="img"`, `aria-label`)
- [x] 2.2 Add `.posture-bar`, `.bar-pass`, `.bar-fail`, `.bar-other` styles to `global.css`
- [x] 2.3 Render `PostureBar` in `PostureCard` between inventory stats and text counts

## 3. Frontend: Evidence recency coloring

- [x] 3.1 Add `freshnessClass` utility function to `posture-view.tsx` (thresholds: <=7d current, <=30d aging, >30d stale, no evidence none)
- [x] 3.2 Add `.freshness-current`, `.freshness-aging`, `.freshness-stale`, `.freshness-none` border-left styles to `global.css`
- [x] 3.3 Apply `freshnessClass` to `PostureCard` article element's class list

## 4. Frontend: Aggregate summary strip

- [x] 4.1 Add `PostureSummary` component to `posture-view.tsx` (reduces `PostureRow[]` to aggregate counts, pass rate, stale count)
- [x] 4.2 Add `isStale` helper (>30d or no evidence)
- [x] 4.3 Add `.posture-summary`, `.summary-stat`, `.summary-stale` styles to `global.css`
- [x] 4.4 Render `PostureSummary` above the posture grid in `PostureView`

## 5. Frontend: Time-preset filter

- [x] 5.1 Add `TimePresets` component to `posture-view.tsx` (buttons: 7d, 30d, 90d, All)
- [x] 5.2 Add `.time-presets` and active-preset styles to `global.css`
- [x] 5.3 Wire `TimePresets` into `PostureView`: on click, set `selectedTimeRange`, re-fetch posture with start/end query params
- [x] 5.4 Update `fetchPosture` to pass `selectedTimeRange` start/end as query params to `GET /api/posture`

## 6. Posture drilldown adjustment

- [x] 6.1 Verify `PostureCard` click target is the "View Details" button only (not the full card surface) now that bar and border are present
