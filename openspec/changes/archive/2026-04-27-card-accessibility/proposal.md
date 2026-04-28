## Why

The QE review identified two Major accessibility gaps across all clickable cards in the workbench:

1. **Posture cards** have `cursor: pointer` on the `<article>` but no `onClick` (fixed separately in `posture-card-clickable`). However, even after that fix, the `<article>` has no `role="button"`, no `tabIndex`, and no keyboard event handler. Screen readers will not announce it as interactive. Keyboard-only users cannot reach or activate it.

2. **Audit cards and inbox cards** already have `onClick` handlers on their `<article>` elements, but they also lack `role`, `tabIndex`, and `onKeyDown` for Enter/Space activation. Same accessibility gap.

3. **Export buttons** (Excel, PDF) are rendered as `disabled` with no explanation. Compliance managers see dead buttons and don't know why.

This affects all four personas — any user navigating with keyboard or assistive technology is blocked.

## What Changes

- **Add `role="button"` and `tabIndex={0}` to all clickable card `<article>` elements** across `posture-view.tsx`, `audit-history-view.tsx`, `draft-review-view.tsx`, and `inbox-view.tsx`.
- **Add `onKeyDown` handler** that triggers the click action on Enter or Space.
- **Add `aria-label`** to cards describing the navigation target (e.g., "View policy ACP-01 details").
- **Add `title` tooltips to disabled export buttons** explaining the limitation (e.g., "Excel export coming soon").
- **Add `title` tooltips to disabled/hidden write actions** for reviewer role (connects to `simple-rbac` change).

## Capabilities

### New Capabilities
- `card-keyboard-navigation`: All clickable cards reachable and activatable via Tab + Enter/Space.
- `card-aria-labels`: Screen readers announce card purpose and target.
- `disabled-button-explanations`: All disabled buttons have `title` attributes explaining why.

### Modified Capabilities
- `posture-card-interaction`: Adds a11y attributes (combined with `posture-card-clickable`).
- `audit-card-interaction`: Adds a11y attributes to existing clickable cards.
- `inbox-card-interaction`: Adds a11y attributes to existing clickable cards.

## Impact

- **Workbench**: Four component files updated with consistent a11y attributes. No behavioral change for mouse users.
- **CSS**: Add `:focus-visible` outline styles for card focus states.
- **Tests**: Add keyboard navigation assertions (Tab to card, Enter to activate).

## Constitution Alignment

### III. Observable Quality

**Assessment**: PASS

Accessibility is observable quality. Users relying on keyboard or assistive technology gain full parity with mouse users.
