## 1. Workspace Store

- [x] 1.1 Create `workbench/src/store/workspace.ts` with `WorkspaceArtifact` type, `Map<string, WorkspaceArtifact>` signal, `activeArtifactName` signal, and CRUD functions (`addArtifact`, `removeArtifact`, `activateArtifact`, `getActiveArtifact`, `getAllArtifacts`)
- [x] 1.2 Add localStorage serialization/deserialization in `workspace.ts` (read on init, write on mutation) using key `complytime-studio-workspace`
- [x] 1.3 Add storage capacity check — warn at 80% of estimated localStorage limit
- [x] 1.4 Refactor `editor.ts` to delegate to `workspace.ts`: `editorContent`, `editorFilename`, `editorDefinition` become computed views of the active workspace artifact
- [x] 1.5 Verify existing components that import from `editor.ts` still compile and function (backward-compat wrappers)

## 2. Tab Bar UI

- [x] 2.1 Create `workbench/src/components/artifact-tabs.tsx` — horizontal tab bar rendering one tab per workspace artifact, with active state styling and close button
- [x] 2.2 Add tab click handler that calls `activateArtifact(name)`
- [x] 2.3 Add tab close handler that calls `removeArtifact(name)` with active-artifact fallback logic
- [x] 2.4 Add horizontal scroll overflow behavior for tab bar
- [x] 2.5 Add CSS for tab bar in `global.css` (tabs, active indicator, close button, scroll)

## 3. Workspace View Integration

- [x] 3.1 Import `artifact-tabs.tsx` into `workspace-view.tsx` and render above the editor
- [x] 3.2 Update `WorkspaceView` Publish button to pass all workspace artifacts (not just active editor content) to `PublishDialog`
- [x] 3.3 Update `handleEditorChange` to write to the active workspace artifact (not raw `editorContent` signal)
- [x] 3.4 Update import flow (`ImportDialog` callback) to add imported artifact to workspace via `addArtifact`

## 4. Agent Proposal → Workspace

- [x] 4.1 Update `applyProposal()` in `editor.ts`/`workspace.ts` to call `addArtifact` (adds to workspace and activates) instead of directly writing `editorContent`
- [x] 4.2 Verify proposal banner Apply/Dismiss still works end-to-end with the new workspace store

## 5. Agent Context Injection

- [x] 5.1 Add `contextArtifacts?: string[]` field to the `Job` interface in `jobs.ts`
- [x] 5.2 Add artifact selection UI to `NewJobDialog` in `jobs-view.tsx` — list workspace artifacts with checkboxes (hidden when workspace is empty)
- [x] 5.3 Store selected artifact names in the `Job.contextArtifacts` field on job creation
- [x] 5.4 Update `streamMessage` call in `chat-drawer.tsx` to serialize selected workspace artifact YAMLs as prefixed text parts in the A2A message (e.g., `--- Context: <name> ---\n<yaml>`)
- [x] 5.5 Add total size check (100 KB cap) with user warning before sending

## 6. Bundle Publish

- [x] 6.1 Update `PublishDialog` props to accept artifacts from workspace store instead of only active editor content
- [x] 6.2 Default all workspace artifacts to checked in the publish dialog
- [x] 6.3 Verify multi-artifact publish works against the existing `/api/publish` endpoint

## 7. Download All

- [x] 7.1 Add "Download All" button to workspace toolbar (visible when > 1 artifact)
- [x] 7.2 Implement download using sequential browser Blob downloads (no zip dependency)
- [x] 7.3 Single-artifact case: download as plain YAML file (no zip)

## 8. Verify

- [x] 8.1 Build passes (TypeScript, Go, Vite production build)
- [ ] 8.2 Manual smoke test: add artifact, switch tabs, close tab, refresh page (localStorage round-trip)
- [ ] 8.3 Manual smoke test: create job with context artifacts, verify agent receives context in chat
- [ ] 8.4 Manual smoke test: publish bundle with multiple workspace artifacts
