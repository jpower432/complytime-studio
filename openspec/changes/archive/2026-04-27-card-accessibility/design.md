# Card accessibility — design

## Context

All clickable cards in the workbench use `<article>` elements with `onClick` handlers and `cursor: pointer` CSS. None have `role`, `tabIndex`, or keyboard event handlers. This is a WCAG 2.1 Level A failure (2.1.1 Keyboard, 4.1.2 Name, Role, Value).

## Decision 1: Shared keyboard handler utility

**Choice:** Create a `cardKeyHandler(callback)` utility that returns an `onKeyDown` handler firing `callback` on Enter or Space (with `preventDefault` for Space to avoid scroll).

**Rationale:** Four components need the same handler. A shared utility prevents divergence.

**Consequences:** One utility function in a shared location (e.g., `lib/a11y.ts`).

## Decision 2: role="button" + tabIndex={0} on all clickable articles

**Choice:** Add `role="button"` and `tabIndex={0}` to every `<article>` that has an `onClick`. Do not convert to `<button>` elements.

**Rationale:** Converting `<article>` to `<button>` would require significant CSS rework (buttons have different default styling). `role="button"` achieves the same accessibility semantics with minimal change.

**Consequences:** Articles appear in tab order. Screen readers announce them as buttons. Visual styling unchanged.

## Decision 3: aria-label with dynamic content

**Choice:** Set `aria-label` on each card to describe its target. Examples:
- Posture card: `"View details for {policy_title}"`
- Audit card: `"View audit for {period}"`
- Inbox draft card: `"Review draft for {policy_id}"`

**Rationale:** Without `aria-label`, screen readers announce the full card text content as the button label, which is noisy and unhelpful.

**Consequences:** Labels must be maintained as card content changes. Keep them concise.

## Decision 4: Focus-visible outline, not focus outline

**Choice:** Use `:focus-visible` (not `:focus`) for card outline styles. This shows the focus ring only for keyboard navigation, not mouse clicks.

**Rationale:** Focus rings on mouse click are visually distracting and unnecessary. `:focus-visible` is the modern standard.

**Consequences:** All modern browsers support `:focus-visible`. No polyfill needed.

## Decision 5: Disabled button tooltips

**Choice:** Add `title` attribute to all `disabled` buttons. Content explains the limitation:
- Export Excel/PDF: `"Coming soon"`
- Write actions for reviewer role: `"Admin role required"`

**Rationale:** A disabled button with no explanation is a dead end. The compliance manager should know whether it's a permission issue or a missing feature.

**Consequences:** `title` attributes are not accessible to all screen reader configurations. As a follow-up, consider `aria-describedby` with a visually hidden span for full coverage.
