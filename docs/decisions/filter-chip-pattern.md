# Filter Chip Pattern

**Date**: 2026-04-26
**Status**: Accepted

## Decision

Active filters render as dismissible chips (`"Field: value ✕"`) across all filterable views in the workbench. This applies whether the filter was applied via click-through navigation, URL parameter, or manual dropdown selection.

## Context

Inventory items, posture cards, and other navigational elements need to set scoped filters on destination views (e.g., clicking a target in inventory should open the evidence tab filtered to that target). Without a visible indicator of the active filter, users get trapped in a filtered view without realizing why data is missing or how to clear the scope.

## Rules

| Rule | Detail |
|:--|:--|
| Chip visibility | Active filter chips appear above the data table, below the filter bar |
| Applied by | Click-through, URL param, dropdown, or any programmatic filter |
| Cleared by | `✕` button on the chip |
| Multiple chips | Supported; combined with AND logic |
| Scope | All filterable views: Evidence, Audit History, any future view |
| Empty state | No chips shown when no filters are active |

## Consequences

- Every view that accepts filters must render active state as chips, not just as selected dropdown values.
- Navigation helpers (e.g., inventory → evidence) set a signal/param; the destination view reads it and renders the chip.
- Filter bar dropdowns and chip state stay in sync — clearing a chip resets the corresponding dropdown.
