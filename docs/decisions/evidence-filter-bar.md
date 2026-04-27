# Evidence Filter Bar

**Date**: 2026-04-26
**Status**: Accepted

## Decision

The evidence filter bar uses a compact layout with always-visible primary filters and an "+ Filter" menu for secondary filters. All active filters render as dismissible chips per the [Filter Chip Pattern](filter-chip-pattern.md).

## Layout

### Always visible

| Control | Type | Rationale |
|:--|:--|:--|
| Policy | Dropdown (populated from API) | Used on nearly every visit |
| Control ID | Text input | Primary drill-down field |
| Date range | Start/end date inputs | Time-scoping is fundamental |
| + Filter | Menu button | Entry point for all other filters |
| Search | Button | Executes query |

### Behind "+ Filter" menu

| Field | Input type | Values |
|:--|:--|:--|
| Target | Dropdown (populated from data) | Dynamic |
| Result | Dropdown | Passed, Failed, Unknown |
| Compliance Status | Dropdown | Compliant, Non-Compliant, Exempt, Not Applicable, Unknown |
| Engine | Dropdown (populated from data) | Dynamic |
| Owner | Dropdown (populated from data) | Dynamic |
| Enrichment Status | Dropdown (populated from data) | Dynamic |

Freshness filtering is handled via the interactive freshness bar (see [Evidence Staleness Model](evidence-staleness-model.md)), not the + Filter menu.

## Interaction

1. Click "+ Filter" — menu lists available fields (excludes already-active filters)
2. Select a field — inline dropdown/input appears
3. Select a value — chip appears, menu closes, query re-executes
4. Click ✕ on chip — filter cleared, query re-executes
5. Multiple chips combine with AND logic

## Extensibility

New evidence fields are added to the "+ Filter" menu only. No layout changes required. Always-visible controls change only with strong usage justification.
