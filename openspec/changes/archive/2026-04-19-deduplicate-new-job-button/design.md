## Context

`JobsView` renders a header with a "+ New Job" button and an empty-state block with a second identical button. Both are disabled when `hasActiveJob()` returns true.

## Goals / Non-Goals

**Goals:**

- Single consistent button placement for job creation.

**Non-Goals:**

- Changing the disabled-when-active behavior (separate concern).
- Redesigning the jobs view layout.

## Decisions

| # | Decision | Rationale |
|:--|:--|:--|
| 1 | Remove empty-state button, keep header button | Header button is always visible regardless of state. Users learn one location. |
| 2 | Empty state becomes text-only | "No active jobs" + descriptive copy. No action needed -- the header button is right above. |

## Risks / Trade-offs

| Risk | Mitigation |
|:--|:--|
| First-time users may not notice the header button | The header button is prominent (primary style) and positioned in the standard action spot (top-right). |
