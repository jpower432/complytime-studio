## Context

The evidence page has minimal filtering (policy, control ID, date range), no cross-view navigation from inventory, and a flat 30-day staleness threshold that ignores policy-defined assessment frequency. The policy detail evidence tab exposes upload controls that should live only on the main evidence management page. Audit history deltas are unlabeled.

Three design decisions are already captured in `docs/decisions/`:
- `filter-chip-pattern.md` — dismissible chips as the universal active-filter indicator
- `evidence-staleness-model.md` — frequency-aware staleness with fallback
- `evidence-filter-bar.md` — compact bar with "+ Filter" menu

## Goals / Non-Goals

**Goals:**
- Unified filter chip component reusable across evidence, audit history, and future views
- Frequency-aware staleness derived from policy YAML at render time
- Interactive freshness bar that doubles as a filter entry point
- Neutral, mode-adaptive row tinting for staleness gradient
- Inventory → evidence cross-navigation with scoped filter
- Read-only evidence tab when embedded in policy detail
- Tooltips on audit history deltas

**Non-Goals:**
- Per-policy configurable thresholds via UI (frequency in YAML is sufficient)
- Server-side staleness computation or materialized freshness columns
- Sorting by freshness (staleness is a visual signal, not a column)
- Refactoring the evidence API query parameters (filters applied client-side from existing response)

## Decisions

### 1. Filter chips as a shared component

A single `FilterChip` component renders any active filter as `"Label: value ✕"`. All filterable views import it. State lives in a `Map<string, string>` signal per view — key is the field name, value is the filter value.

Alternative: per-field signals (e.g., `selectedTargetId`, `selectedEvalResult`). Rejected — doesn't scale to N filters and requires new signals for every new field.

### 2. Client-side frequency parsing

The policy `content` column is already fetched for `PolicyDetailView`. Parse the YAML to build a `Map<string, number>` from `requirement_id → cycle_days`. Evidence rows join against this map at render time.

Alternative: backend endpoint that returns enriched evidence with expected frequency. Rejected — adds API surface, the policy content is already available client-side, and the mapping is lightweight.

### 3. Freshness bar as filter entry point

The freshness bar is a `<div>` with four proportional segments. Each segment has an `onClick` that creates a freshness filter chip. No counts or percentages rendered — proportions are the data. Tooltip on hover shows bucket name.

Alternative: separate freshness dropdown in the "+ Filter" menu. Rejected — the bar is more visual and provides at-a-glance health without interaction.

### 4. Neutral tint palette

Row backgrounds use HSL-based neutral shades at low opacity. CSS custom properties switch values between light and dark mode via `prefers-color-scheme`. No primary colors.

```
--freshness-current:    hsl(210, 15%, 88%)  / hsl(180, 8%, 30%)
--freshness-aging:      hsl(30, 10%, 75%)   / hsl(30, 8%, 42%)
--freshness-stale:      hsl(220, 8%, 58%)   / hsl(220, 6%, 58%)
--freshness-very-stale: hsl(220, 10%, 35%)  / hsl(220, 5%, 78%)
```

Applied as `background-color` at 8% opacity on table rows.

### 5. "+ Filter" menu implementation

A button that toggles a dropdown listing available fields. Selecting a field shows an inline value selector (dropdown for enum fields, populated-from-data dropdown for dynamic fields). Selecting a value creates a chip, closes the menu, and triggers a re-filter. Fields already active as chips are excluded from the menu.

### 6. Embedded read-only gate

`EvidenceView` already has `const embedded = !!policyIdOverride`. The upload button condition changes from `role === "admin"` to `!embedded && role === "admin"`. No new prop needed.

## Risks / Trade-offs

- **[YAML parsing in browser]** Policy content is a string blob. Parsing YAML client-side adds a dependency or requires a lightweight parser. → Mitigation: the content is already valid YAML; a minimal parser for `assessment-plans` extraction is sufficient. Evaluate `yaml` package size; if too heavy, parse with regex for the narrow `frequency` field.
- **[Stale threshold mismatch]** Agent posture-check and UI may compute slightly different staleness if frequency mappings diverge. → Mitigation: share the frequency-to-days mapping as a constant in `freshness.ts`, document it as the canonical source.
- **[Client-side filtering at scale]** Filtering 200 rows client-side is fine. If evidence volume grows beyond the current `limit=200`, server-side filtering will be needed. → Mitigation: current cap is acceptable; flag for revisit if limit increases.
