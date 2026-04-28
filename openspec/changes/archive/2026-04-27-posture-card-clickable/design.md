# Posture card clickable — design

## Context

`PostureCard` in `posture-view.tsx` renders an `<article class="posture-card">` with CSS `cursor: pointer` and `:hover` border highlight, but `onClick` is only on a nested `<button>`. All other clickable cards in the workbench wire `onClick` to the `<article>` directly.

## Decision 1: Full-card click, drop the button

**Choice:** Add `onClick={() => navigateToPolicy(row.policy_id)}` to the `<article>` element. Remove the `<button class="posture-drilldown-btn">`.

**Rationale:** Matches the established pattern in `audit-history-view.tsx` (line 126), `draft-review-view.tsx` (line 287), and `inbox-view.tsx` (line 267). Users already expect the card to be the click target because of the CSS affordance.

**Consequences:** No behavior change for users who clicked the button. Users who clicked the card body (and got nothing) now get the expected navigation.

## Decision 2: Keep CSS as-is

**Choice:** `.posture-card` already has `cursor: pointer` and `:hover` border. No CSS changes needed beyond removing `.posture-drilldown-btn`.

**Rationale:** The visual affordance was always correct. Only the interaction was broken.
