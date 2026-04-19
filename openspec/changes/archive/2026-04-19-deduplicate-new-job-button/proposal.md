## Why

The Jobs view renders two identical "+ New Job" buttons: one in the header (always visible) and one in the empty-state CTA. Both are disabled when a job is active, so having two provides no additional access. The duplication is visually noisy and violates the convention-over-configuration principle -- the user shouldn't wonder which button to click.

## What Changes

- Remove the "+ New Job" button from the empty-state section of the Active area.
- Keep the header button as the single, consistent entry point for job creation.
- Update empty-state copy to be informational only (no action).

## Capabilities

### New Capabilities

_(none)_

### Modified Capabilities

- `job-lifecycle`: The "Empty active state" scenario changes -- removes the requirement for a prominent New Job button in the empty state.

## Impact

- **`workbench/src/components/jobs-view.tsx`**: Remove the button from the empty-state block in `JobsView`.
