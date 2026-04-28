# Tasks — card accessibility

## Shared utility

- [x] Create `workbench/src/lib/a11y.ts` with `cardKeyHandler(callback: () => void)` returning an `onKeyDown` handler that fires `callback` on Enter or Space (with `e.preventDefault()` for Space).

## Posture cards

- [x] In `posture-view.tsx` `PostureCard`: add `role="button"`, `tabIndex={0}`, `onKeyDown={cardKeyHandler(...)}`, `aria-label={`View details for ${row.title}`}` to the `<article>`.

## Audit history cards

- [x] In `audit-history-view.tsx`: add `role="button"`, `tabIndex={0}`, `onKeyDown={cardKeyHandler(...)}`, `aria-label` to each `<article class="audit-card">`.

## Draft review cards

- [x] In `draft-review-view.tsx`: add `role="button"`, `tabIndex={0}`, `onKeyDown={cardKeyHandler(...)}`, `aria-label` to each `<article class="audit-card">`.

## Inbox cards

- [x] In `inbox-view.tsx`: add `role="button"`, `tabIndex={0}`, `onKeyDown={cardKeyHandler(...)}`, `aria-label` to each `<article class="inbox-card">` (both draft and notification variants).

## Focus styles

- [x] Add `.posture-card:focus-visible, .audit-card:focus-visible, .inbox-card:focus-visible` CSS rule with `outline: 2px solid var(--accent); outline-offset: 2px;` to `global.css`.

## Disabled button tooltips

- [x] Add `title="Coming soon"` to disabled Export Excel and Export PDF buttons in `requirement-matrix-view.tsx`.
- [ ] Add `title="Admin role required"` to write-action elements hidden/disabled for reviewer role. Coordinate with `simple-rbac` frontend tasks.

## Tests

- [ ] Verify Tab reaches posture card, Enter activates navigation.
- [ ] Verify Tab reaches audit card in History, Enter activates navigation.
- [ ] Verify Tab reaches inbox card, Enter activates navigation.
- [ ] Verify Space activates cards without scrolling the page.
- [ ] Verify focus-visible ring appears on keyboard focus, not on mouse click.
- [ ] Verify disabled export buttons have visible tooltip on hover.
