## 1. Workspace Store

- [x] 1.1 Add `gemaraVersion: string` field to the artifact type in `workspace.ts` (default `"latest"`)
- [x] 1.2 Add `updateActiveGemaraVersion(version: string)` mutation
- [x] 1.3 Expose `editorGemaraVersion` computed signal in `editor.ts`

## 2. Toolbar UI

- [x] 2.1 Add version text input to the artifact toolbar in `workspace-view.tsx` (next to definition dropdown)
- [x] 2.2 Wire input value to `editorGemaraVersion` / `updateActiveGemaraVersion`
- [x] 2.3 Pass `editorGemaraVersion` to `validate()` call in `handleValidate`

## 3. Styling

- [x] 3.1 Add `.version-input` styles to `global.css` (narrow input, monospace font, matches definition dropdown height)
