## Context

The posture view (`posture-view.tsx`) renders a card grid from `GET /api/posture`. Each card shows text counts (passed/failed/other), a pass rate percentage, relative timestamps, risk badges, and inventory stats. The API returns `PostureRow` structs from a ClickHouse query joining `policies` and `evidence` — no time filtering, no visual density beyond text.

Three views already consume `selectedTimeRange` signals for time-filtered queries (requirement matrix, audit history, evidence). The posture view does not.

## Goals / Non-Goals

**Goals:**
- Increase scan speed for analysts triaging posture across multiple policies.
- Surface evidence staleness visually without requiring timestamp reading.
- Provide a cross-policy aggregate so analysts see overall health at a glance.
- Enable time-filtered posture to answer "how did we look in Q1?" without navigating to the requirement matrix.

**Non-Goals:**
- Graph/chart libraries. All visuals use native HTML/CSS (divs, borders, custom properties).
- Temporal slider or dual-range input. Time presets (7d/30d/90d/All) cover the primary use cases.
- New API endpoints. The existing `GET /api/posture` gains optional query parameters.
- Changes to the requirement matrix, evidence, or audit history views.

## Decisions

**1. Stacked bar via CSS flex, not SVG or canvas.**

Native `<div>` segments inside a flex container. Each segment's width is a percentage of total. Accessible via `role="img"` and `aria-label`.

Alternative: SVG `<rect>` elements. Rejected — adds complexity for a 6px-tall bar. CSS flex handles this natively with less markup.

**2. Recency coloring via card border-left, not background tint.**

A 3px left border colored by freshness band (current/aging/stale/none). Thresholds: <=7d current, <=30d aging, >30d stale, no evidence = none.

Alternative: Background color tint. Rejected — conflicts with light/dark theme surface colors and reduces text contrast. Border is additive, not destructive.

**3. Time presets via relative buttons, not date inputs.**

Four buttons (7d / 30d / 90d / All) that compute `start`/`end` relative to now and write to `selectedTimeRange`. The posture API receives these as query parameters.

Alternative: Reuse the date `<input>` pattern from requirement matrix. Rejected — adds clutter to the posture view for a feature that's primarily "show me recent." Presets are one click vs four (click input, pick start, click input, pick end).

**4. Backend: optional `start`/`end` on `ListPosture`, conditional WHERE clause.**

`ListPosture(ctx, start, end time.Time)` adds `AND e.collected_at >= ? AND e.collected_at <= ?` when non-zero. The handler reads `start` and `end` query params, parses as RFC 3339 or date-only, and passes to the store.

Alternative: New endpoint `GET /api/posture/filtered`. Rejected — unnecessary API surface growth for optional query parameters on an existing endpoint.

**5. Aggregate summary strip as a computed component, not a new API.**

`PostureSummary` reduces the same `PostureRow[]` array already fetched. No additional API call. Cross-policy pass rate, total counts, stale count, and a full-width stacked bar.

Alternative: Server-side aggregate endpoint. Rejected — the posture response already contains all the data. Client-side reduce avoids an extra round-trip.

## Risks / Trade-offs

**Freshness thresholds are hardcoded (7d/30d).** Different policies may have different assessment cadences. A monthly-assessed policy looks "aging" at day 8.
→ Acceptable for v1. If needed later, thresholds can move to policy metadata without changing the visual pattern.

**Time-filtered posture changes card counts.** Filtering to 7d may show "0 passed" for a policy with infrequent evidence, which could alarm analysts.
→ Mitigate by showing the active time filter prominently and defaulting to "All" (no filter). The preset buttons make it clear a filter is active.

**`PostureBar` at 6px height may be hard to distinguish on very small differences** (e.g., 98% pass vs 100%).
→ Acceptable — the text counts remain visible below the bar. The bar communicates ratio at a glance; exact numbers are in the text.
