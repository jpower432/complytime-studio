# Balanced Color Palette

**Status:** Accepted
**Date:** 2026-04-27

## Context

The workbench originally used Tailwind-derived saturated primaries (`#22c55e`, `#ef4444`, `#f59e0b`, `#0891b2`). An initial correction over-desaturated to muddy tones (`#4a8c6a`, `#a05555`, `#b08a45`) that lost at-a-glance readability on dense dashboards.

## Decision

Use **medium-saturation colors** — vibrant enough to read at a glance, professional enough for extended use. Centralized as CSS custom properties in `:root`.

| Semantic | Light | Dark | Purpose |
|:--|:--|:--|:--|
| Accent | `#3b8ea5` | `#4db8d1` | Interactive elements, active states |
| Pass/Strength | `#2e8b57` | `#4dc78a` | Sea green for positive indicators |
| Finding/Warning | `#c08b30` | `#d4a84a` | Warm amber for attention items |
| Gap/Error | `#c0392b` | `#d95b4e` | Brick red — clearly signals danger |
| Observation | `#7068a6` | `#9490cc` | Rich purple for informational |

## Rationale

- **Readability**: Medium saturation preserves semantic meaning without neon glare or muddy ambiguity.
- **Professional tone**: Aligned with data dashboard conventions (Grafana, GitHub).
- **Centralization**: All hardcoded hex values replaced with `var(--color-*)` tokens. One change propagates everywhere.
- **Dark theme parity**: Dark variants are lighter/warmer to maintain contrast on dark backgrounds.

## Alternatives Considered

| Approach | Rejected Because |
|:--|:--|
| Tailwind primaries | Too saturated for data-dense dashboards |
| Fully desaturated | Washed out, lost at-a-glance readability |
| Grayscale only | Loses semantic color meaning (pass/fail/warn) |
