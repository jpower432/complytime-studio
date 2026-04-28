# Tasks — posture card clickable

## Component change

- [x] In `posture-view.tsx` `PostureCard`: add `onClick={() => navigateToPolicy(row.policy_id)}` to the `<article class="posture-card">` element.
- [x] Remove the `<button class="posture-drilldown-btn">View Details</button>` and its wrapping markup.

## CSS cleanup

- [x] Remove `.posture-drilldown-btn` rule from `global.css`.

## Tests

- [ ] Update any E2E or integration selectors that target `.posture-drilldown-btn` to target `article.posture-card` click instead.
- [ ] Verify clicking the posture card body navigates to `#/posture/{policy_id}`.
