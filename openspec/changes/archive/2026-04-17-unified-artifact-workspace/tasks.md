## 1. Editor State Store

- [x] 1.1 Create `workbench/src/store/editor.ts` with module-level signals: `editorContent`, `editorFilename`, `editorDefinition`, `mappingReferences`
- [x] 1.2 Add `setEditorArtifact(name, yaml, definition)` helper that updates all three signals atomically
- [x] 1.3 Add `injectMappingReference(ref)` helper that appends to `mappingReferences` signal and patches `editorContent` YAML string

## 2. Mapping-Reference YAML Injection

- [x] 2.1 Create `workbench/src/lib/yaml-inject.ts` with `injectMappingRef(yaml: string, ref: MappingReference): string` — string-level insertion that finds `mapping-references:` block or creates one under `metadata:`
- [x] 2.2 Handle case: `mapping-references:` exists with entries — append after last entry
- [x] 2.3 Handle case: `metadata:` exists but no `mapping-references:` — insert key with new entry
- [x] 2.4 Handle case: no `metadata:` block — prepend minimal metadata scaffold
- [x] 2.5 Add `extractMappingRefFromYaml(yaml: string): MappingReference | null` — parses imported artifact to extract `metadata.id`, `metadata.version`, `title`, `metadata.description`

## 3. Workspace Editor View

- [x] 3.1 Create `workbench/src/components/workspace-view.tsx` — reads from editor signals, renders `YamlEditor`, toolbar (Validate, Save, Publish, Import), and validation/save feedback
- [x] 3.2 Wire Validate button to existing `/api/validate` using `detectDefinition` on editor content
- [x] 3.3 Wire Save button to existing `saveToWorkspace` using `editorFilename` signal
- [x] 3.4 Wire Publish button to open `PublishDialog` with current editor content as a single artifact
- [x] 3.5 Wire Import button to open the import dialog (task group 5)

## 4. Chat Drawer

- [x] 4.1 Create `workbench/src/components/chat-drawer.tsx` — wraps existing `ChatPanel` in a slide-in drawer with close/open toggle
- [x] 4.2 Show drawer automatically when a mission enters active state; hide when no active mission
- [x] 4.3 Add "Chat" toggle button to workspace toolbar — visible when an active mission exists
- [x] 4.4 Wire `handleReply` in drawer to `sendReply` using mission's stored `agentName`

## 5. Registry Import Dialog

- [x] 5.1 Refactor `registry-browser.tsx` into `workbench/src/components/import-dialog.tsx` — wrap existing state machine in a dialog overlay
- [x] 5.2 In the layer view phase, detect if content is a Gemara artifact using `isGemaraArtifact` from `artifact-detect.ts`
- [x] 5.3 If Gemara artifact detected, show "Import Reference" button alongside "Save to Workspace"
- [x] 5.4 On "Import Reference" click: call `extractMappingRefFromYaml` on layer content, add OCI URL from registry/repo/tag coordinates, call `injectMappingReference` on editor store, close dialog

## 6. App Routing and Navigation

- [x] 6.1 Update `app.tsx` — change default view from `"missions"` to `"workspace"`; add `"workspace"` to View type
- [x] 6.2 Update `sidebar.tsx` — reorder nav: Workspace (primary/default), Missions
- [x] 6.3 Remove standalone `RegistryBrowser` from app routing (replaced by import dialog)
- [x] 6.4 Update mission creation flow — after `createMission` + `navigate`, go to `"workspace"` instead of `"detail"` so the editor is visible with chat drawer

## 7. Connect Missions to Editor

- [x] 7.1 In `MissionDetail` (or its replacement), when `onArtifact` / `onMessage` extracts a Gemara artifact, call `setEditorArtifact` on the editor store
- [x] 7.2 Remove standalone `MissionDetail` view — its chat functionality moves to the chat drawer, its artifact display is now the workspace editor
- [x] 7.3 Ensure `streamTask` SSE subscription starts when mission is created and persists across workspace view (not tied to a component mount/unmount cycle)
