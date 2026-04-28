# Proposal: Audit History UX

## Problem

Audit history cards were changed from expandable inline cards (with reasoning visible on expand) to flat clickable rows that navigate to a full-page workspace. This created two issues:

1. **Clunky navigation**: Users must leave the policy context to review a single audit. Scanning multiple audits requires repeated back-navigation.
2. **Redundant workspace panel**: The audit workspace embedded full Requirements, Evidence, and History views in a right panel — duplicating what Policy Detail already provides.

## Solution

1. **Restore expandable inline cards** in the audit history list. Click to expand/collapse. Expanded state shows full YAML content and agent reasoning. "Open Workspace" button available for heavy tasks (draft editing, promote).
2. **Simplify workspace right panel** from a 3-tab context clone to a compact summary sidebar with audit metadata, result stats, and deep links back to Policy Detail tabs.

## User Personas

- **Auditor**: Scans historical audits for gaps and reasoning. Needs expand-in-place, not page navigation.
- **Compliance Manager**: Reviews audit outcomes to coordinate remediation. Needs at-a-glance verdict mix without leaving the policy view.

## Scope

| In Scope | Out of Scope |
|:--|:--|
| `audit-history-view.tsx` — expandable cards | Audit workspace result cards (unchanged) |
| `audit-workspace-view.tsx` — simplified right panel | Draft review workflow (unchanged) |
| CSS for expand/collapse animation | New API endpoints |
