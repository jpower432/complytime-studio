## Context

The workbench currently has three top-level views behind a sidebar: Missions, Mission Detail, and Registry. The YAML editor (CodeMirror) lives exclusively inside `ArtifactPanel`, which is a child of `MissionDetail`. The registry browser is a self-contained state machine (`input → repos → tags → manifest → layer`) that renders layer content in a `<pre>` tag with no editing capability.

Current component tree:

```
App
├── Sidebar [Missions | Registry]
├── MissionsView (list + new mission dialog)
├── MissionDetail
│   ├── ChatPanel (streaming agent messages)
│   └── ArtifactPanel
│       ├── YamlEditor (CodeMirror) ← only editor instance
│       ├── Validate / Save / Publish toolbar
│       └── PublishDialog
└── RegistryBrowser (standalone, no editor connection)
```

The prior change (`fix-workbench-e2e-flow`) fixed the mission dialog, agent picker, registry JSON errors, and added workspace save. This change restructures the view hierarchy so the editor is central.

## Goals / Non-Goals

**Goals:**
- Editor is the default view — users land on it when they open the workbench
- Missions feed artifacts into the editor (chat becomes a supporting panel)
- Registry import injects `mapping-references` into the active editor document
- Single editing experience regardless of how content arrived

**Non-Goals:**
- Multi-document editing (tabs across multiple artifacts open simultaneously) — keep single-document focus for now
- Server-side editor state — editor state stays in the browser (localStorage + signals)
- Editing the imported artifact itself — import only injects the reference pointer
- Full registry browser as a standalone view — it becomes an import dialog

## Decisions

### D1: New view hierarchy — editor-centric layout

**Decision:** Restructure the app to three views: `workspace` (default), `missions`, and `detail`. The workspace view contains the YAML editor, toolbar, and import panel. Mission detail retains chat but delegates its artifact display to the shared workspace editor.

```
App (after)
├── Sidebar [Workspace | Missions]
├── WorkspaceView (DEFAULT)
│   ├── EditorToolbar [Validate | Save | Publish | Import]
│   ├── YamlEditor (CodeMirror)
│   ├── ImportDialog (registry browser, triggered by Import button)
│   └── MissionChatDrawer (slides in when a mission is active)
└── MissionsView (list + new mission dialog)
```

**Rationale:** The editor should be visible and usable without starting a mission. Missions become a way to *populate* the editor, not the only path to it. The chat drawer slides in when a mission is active and hides when there's no active mission or the user dismisses it.

**Alternative considered:** Keep three top-level views and share editor state via signals. Rejected — the user would navigate between views constantly, and the editor would unmount/remount losing CodeMirror state.

### D2: Shared editor state via Preact signals

**Decision:** Create a shared `editorStore` with signals for the active document:

```typescript
// store/editor.ts
export const editorContent = signal("");
export const editorFilename = signal("artifact.yaml");
export const editorDefinition = signal("#ThreatCatalog");
export const mappingReferences = signal<MappingReference[]>([]);
```

All sources (mission artifacts, registry imports, file loads) write to these signals. The `WorkspaceView` subscribes to them.

**Rationale:** Signals are already the app-level state pattern (see `currentView`, `missionsList`). A signal-based store lets any component — mission detail, import dialog, sidebar — update the editor without prop drilling. Unlike the broken signals-in-function-components pattern fixed in the prior change, these are module-level singletons that persist across renders.

**Alternative considered:** React context / useState lifted to App. More boilerplate, no benefit over module-level signals for a single-editor app.

### D3: Mission artifacts populate the editor automatically

**Decision:** When `MissionDetail` receives an artifact via SSE (`onArtifact` / `onMessage` with extracted YAML), it writes to the shared editor signals in addition to the mission store. The editor updates live as the agent produces content.

**Rationale:** The user expects to see YAML appear in the editor as the agent works. The mission store continues to persist artifacts for history. The editor signals are the "live" view.

**Conflict resolution:** If the user has made local edits in the editor and the agent produces a new artifact, the agent's output wins (overwrites). The user's edits are in the CodeMirror undo buffer and can be recovered with Ctrl+Z. This matches the mental model — the agent is generating, the user is refining after.

### D4: Registry import dialog replaces standalone browser

**Decision:** The registry browser becomes a modal dialog triggered by an "Import" button in the editor toolbar. It reuses the existing state machine (`input → repos → tags → manifest → layer`). When the user reaches the layer view and the content is a Gemara artifact, an "Import Reference" button appears. Clicking it:

1. Parses the layer YAML to extract `metadata.id`, `metadata.version`, `title`, and `metadata.description`
2. Constructs the OCI URL from `{registry}/{repo}:{tag}`
3. Appends a `mapping-references` entry to the editor's active document
4. Closes the dialog

```yaml
# Injected into the editor document's metadata.mapping-references:
- id: SEC.SLAM.CM
  title: Container Management Tool Security Threat Catalog
  version: "1.0.0"
  url: ghcr.io/complytime/sec-slam-cm-threats:v1.0.0
  description: |
    Threat catalog for container management tool security assessment
```

**Rationale:** The registry browser's value is as an import source, not a standalone view. Making it a dialog keeps the editor visible underneath. The existing state machine works as-is — only the final action changes from "view in `<pre>`" to "parse + inject reference."

**Alternative considered:** Side panel instead of dialog. Viable future enhancement, but dialog is simpler and doesn't require responsive layout work.

### D5: YAML mapping-reference injection strategy

**Decision:** Use string-level YAML manipulation rather than full parse-and-serialize. Find the `mapping-references:` key in the editor content, locate the end of its list (by indentation), and insert the new entry. If `mapping-references:` doesn't exist, insert the entire block under `metadata:`.

**Rationale:** Full YAML parse → modify → serialize would lose comments, formatting, and ordering that the user or agent carefully crafted. String-level insertion preserves the document as-is and only adds content. This is the same approach used by `extractArtifacts` in `artifact-detect.ts` — regex-based detection, targeted modification.

**Fallback:** If the document has no `metadata:` block at all, prepend a minimal metadata scaffold with the reference.

### D6: Chat drawer for active missions

**Decision:** When a mission is active (`submitted`, `working`, `input-required`), a chat drawer slides in from the right side of the workspace view. It contains the existing `ChatPanel` component. The user can dismiss it to focus on the editor and re-open it from the toolbar.

**Rationale:** Chat is essential during a mission but irrelevant otherwise. A drawer keeps it accessible without permanently consuming screen space. The editor remains full-width when no mission is active.

## Risks / Trade-offs

| Risk | Mitigation |
|:--|:--|
| String-level YAML injection could corrupt documents with unusual formatting | Validate after injection; if validation fails, show the reference in a copyable block and let the user paste manually |
| Agent overwrites user edits when new artifact arrives | CodeMirror undo buffer preserves user work; add a toast notification "Agent updated artifact" so the overwrite isn't silent |
| Registry import dialog blocks editor interaction | Dialog is dismissible; editor state persists underneath; user can cancel and return anytime |
| Removing registry as standalone view reduces discoverability | Import button in toolbar is prominent; sidebar still has a clear entry point |
| CodeMirror state loss during view transitions | Editor lives in the default view and is never unmounted; mission navigation uses the chat drawer, not a view switch |
