## Why

The `validate()` API already supports a `version` parameter, but the UI hardcodes it to `"latest"`. Users authoring artifacts against a specific Gemara release (e.g., `0.20.0`) have no way to validate against that version. This is a gap for any team maintaining artifacts pinned to a release.

## What Changes

- Add a version input to the artifact toolbar (next to the definition dropdown).
- Default to `latest`. User can type a specific version string (e.g., `0.20.0`).
- Pass the user-specified version through to `validate()` and the gateway.
- Store the selected version per-artifact in the workspace store.

## Capabilities

### New Capabilities

- `gemara-version-select`: Toolbar control for choosing the Gemara schema version used during validation.

### Modified Capabilities

_(none -- the validate API already accepts `version`; no backend change needed)_

## Impact

- **`workbench/src/components/workspace-view.tsx`**: New input element in the toolbar.
- **`workbench/src/store/workspace.ts`**: New `gemaraVersion` field per artifact.
- **`workbench/src/store/editor.ts`**: Expose `editorGemaraVersion` signal.
