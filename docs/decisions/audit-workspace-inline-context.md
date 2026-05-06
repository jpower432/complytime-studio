# Audit Workspace Inline Context

**Status:** Accepted
**Date:** 2026-05-05

## Context

The Audit Review workspace (`#/reviews/{id}`) displays the agent's opinion (result cards with classification + reasoning) but requires the reviewer to navigate away to see the underlying requirement text or supporting evidence. This breaks the reviewer's flow — they must context-switch between three separate pages to validate a single result.

The original design separated concerns across views:

| View | Shows |
|:-----|:------|
| Requirements tab | What the policy demands |
| Evidence page | What data exists |
| Audit Review | Agent's opinion |

Reviewers need all three in one place to make an informed accept/override decision without navigation.

## Decision

**Inline requirement text and evidence directly in each Audit Review result card.**

Each result card now shows:

1. **Requirement text** — fetched from the requirements API using the result's control_id, displayed below the result title
2. **Evidence rows** — collapsible section per card, lazy-loaded from `fetchRequirementEvidence` on expand
3. **Opinion** — existing classification, description, agent reasoning, reviewer override (unchanged)

No new API endpoints required. Uses existing `fetchRequirementMatrix` and `fetchRequirementEvidence` from `api/requirements.ts`.

## Consequences

- Reviewer never leaves the workspace to validate a result.
- Evidence fetch is lazy (per-card expand) to avoid loading all evidence upfront for large audits.
- Requirement text map is fetched once per audit load (one API call).
- Result card height increases; scrollable evidence section prevents layout blow-up.
- Maintains the "single source of truth" principle — Evidence page remains canonical for full evidence search/filter; the workspace provides a scoped read-only slice.
