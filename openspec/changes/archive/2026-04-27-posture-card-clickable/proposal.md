## Why

The posture card has `cursor: pointer` and a hover border effect on the `<article>` element, but the only click handler is on a "View Details" button nested inside. Users click the card body and nothing happens. Every other card in the workbench (audit cards, inbox cards, draft cards) navigates on full-card click. The inconsistency erodes trust in the UI.

The Developer/SRE persona flagged this in QE review: "the card lies to the user."

## What Changes

- **Remove the "View Details" button** from `PostureCard` in `posture-view.tsx`.
- **Move `onClick={() => navigateToPolicy(row.policy_id)}` to the `<article>` element**, matching the pattern used by `audit-card` in `audit-history-view.tsx` and `inbox-card` in `inbox-view.tsx`.
- **Remove `.posture-drilldown-btn` CSS rule** (`margin-top: 8px; width: 100%`).

## Capabilities

### Modified Capabilities
- `posture-card-interaction`: Card body becomes the click target; button removed. Navigation target unchanged (`#/posture/{policy_id}`).

## Impact

- **Workbench**: `posture-view.tsx` — one prop change, one element removal. `global.css` — one rule removal.
- **Tests**: Update any selectors targeting `.posture-drilldown-btn`. Add assertion that clicking `article.posture-card` navigates to drill-down.

## Constitution Alignment

### IV. Testability

**Assessment**: PASS

Single interaction change, trivially testable with a click event on the article element.
