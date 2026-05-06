# View Single Source of Truth

**Date**: 2026-05-04
**Status**: Accepted

## Decision

Each data domain (Evidence, Inventory) has exactly one canonical view. Other views act as **launchpads** that navigate to the canonical view with filter chips pre-set.

## Context

The workbench has multiple entry points to the same data: policy detail tabs, dashboard cards, posture cards, inventory rows. Embedding full sub-views (e.g., evidence table inside policy detail tabs) duplicates state management, creates competing filter UIs, and makes it unclear which view is authoritative.

## Rules

| Rule | Detail |
|:--|:--|
| Canonical views | Evidence view owns evidence. Inventory view owns inventory. |
| No embedded duplicates | Policy detail, program detail, and dashboard do NOT embed full evidence/inventory tables as tabs |
| Launchpad pattern | Context views provide buttons/links that navigate to the canonical view |
| Filter preset | Launchpads set filter chips on the destination (e.g., Policy chip, Program chip, Target chip) |
| Chip visibility | Per [Filter Chip Pattern](filter-chip-pattern.md), preset filters render as dismissible chips |

## Canonical Views

| Domain | Canonical Route | View Component |
|:--|:--|:--|
| Evidence | `#/evidence` | `EvidenceView` |
| Inventory | `#/inventory` | `InventoryView` |
| Requirements | embedded in Policy Detail | `RequirementMatrixView` |
| Audit History | embedded in Policy Detail | `AuditHistoryView` |

Requirements and Audit History are policy-intrinsic — they only exist in the context of a specific policy, so they remain as tabs in the policy detail view.

## Policy Detail Tabs

| Tab | Behavior |
|:--|:--|
| Requirements | Renders inline (policy-intrinsic data) |
| History | Renders inline (policy-intrinsic data) |
| Inventory | Launchpad → navigates to `#/inventory` with Policy chip |
| Evidence | Launchpad → navigates to `#/evidence` with Policy chip |

## Consequences

- Simpler state: each domain has one filter implementation, one URL scheme, one empty state.
- Deep links from notifications, dashboard, and chat always land on the same view.
- Policy detail stays focused on policy-intrinsic concerns (requirements + audit history).
