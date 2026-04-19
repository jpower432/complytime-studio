## Why

The artifact toolbar has 7-9 buttons on a single row depending on state. This crowds the UI and makes primary actions (Validate, Publish) compete for attention with secondary actions (Copy, Download, Import). The problem worsens as the workspace grows (Download All appears conditionally, Chat toggle appears during jobs).

## What Changes

- Keep only primary actions visible in the toolbar: Type dropdown, Validate, Publish, Chat toggle.
- Move secondary actions (Copy YAML, Download YAML, Download All, Import) behind a `...` overflow menu.
- Add click-outside-to-dismiss behavior for the overflow menu.
- No functional changes -- all actions remain accessible, just reorganized.

## Capabilities

### New Capabilities

_(none)_

### Modified Capabilities

- `workspace-editor`: Toolbar layout changes from flat button row to primary + overflow split. All existing actions remain; their placement changes.

## Impact

- **`workspace-view.tsx`**: Toolbar JSX restructured. New state for menu open/close. `useRef` + `useEffect` added for click-outside handling.
- **`global.css`**: New `.toolbar-overflow`, `.toolbar-overflow-menu`, `.toolbar-overflow-item` styles.
- **UX**: Fewer visible buttons reduces cognitive load. Secondary actions one click further away.
