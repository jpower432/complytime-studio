## Context

The artifact toolbar in `workspace-view.tsx` renders all actions as a flat row of buttons. As features were added (Copy, Download All, Import, Chat toggle), the row grew beyond a comfortable density.

## Goals / Non-Goals

**Goals:**

- Reduce visible toolbar items to 3-4 primary actions.
- Group secondary file operations behind an overflow menu.
- Maintain full keyboard and mouse accessibility.

**Non-Goals:**

- Responsive/mobile layout (desktop-only SPA).
- Keyboard shortcuts for overflow items (future enhancement).
- Changing which actions exist -- purely a layout reorganization.

## Decisions

| # | Decision | Rationale |
|:--|:--|:--|
| 1 | Overflow menu (not icon buttons or two rows) | Standard IDE pattern. Scales as we add features. Smallest code change. |
| 2 | Click-outside-to-close via `useEffect` + `mousedown` listener | Lightweight, no external dependency. Cleanup on unmount via return function. |
| 3 | SVG three-dot icon inline (not icon library) | Avoids adding a dependency for a single icon. |
| 4 | `position: absolute` dropdown anchored right | Prevents menu from overflowing the editor area when toolbar is narrow. |

## Risks / Trade-offs

| Risk | Mitigation |
|:--|:--|
| Secondary actions less discoverable | Three-dot is a well-known affordance. Tooltip reads "More actions". |
| Extra click for Copy/Download | Power users can use Ctrl+C in editor. Download is infrequent. |
