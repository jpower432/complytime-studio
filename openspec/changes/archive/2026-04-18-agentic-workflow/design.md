## Context

ComplyTime Studio is an agentic platform for GRC artifact authoring and review. Agents drive the work; users direct, review, approve, and ship. Today the platform holds exactly one artifact at a time in a singleton editor signal (`editor.ts`). Each job is isolated — output from one agent cannot feed another. Publishing operates on that single artifact.

The user's stated vision: "Cursor for GRC." The workflow gap is agent chaining and multi-artifact management. Agents already stream well; the editor already supports proposals with undo. The missing layer is a workspace that accumulates artifacts across jobs and feeds them forward.

## Goals / Non-Goals

**Goals:**
- Replace singleton editor signal with a multi-artifact workspace store
- Allow users to select workspace artifacts as input context for new jobs
- Enable bundle publish of multiple artifacts in one operation
- Persist workspace state across page refreshes (localStorage)
- Maintain all existing single-artifact UX (validate, download, copy, import)

**Non-Goals:**
- Server-side workspace persistence (future: Git-backed)
- Real-time collaborative editing
- Automatic agent chaining (agents don't invoke other agents autonomously)
- File-system workspace (no `.complytime/artifacts/` directory — pure client state for now)
- Multi-tab split-pane editor (one artifact active at a time, tabs for switching)

## Decisions

### D1: Workspace store replaces editor store

**Choice:** New `workspace.ts` signal store. Holds `Map<string, WorkspaceArtifact>` keyed by artifact name. One artifact is "active" (shown in the editor). The existing `editorContent`, `editorFilename`, `editorDefinition` signals remain but are computed from the active artifact.

**Why not extend editor.ts:** The editor store is a bag of independent signals. A workspace needs a coherent collection with selection state. Wrapping the editor signals would add indirection without simplifying the model.

**Migration:** `editor.ts` exports become thin wrappers around `workspace.ts` to avoid breaking all import sites at once. Existing components continue using `editorContent` etc. until incrementally updated.

### D2: Artifact tabs for switching

**Choice:** Horizontal tab bar above the CodeMirror editor. Each tab shows the artifact name. Clicking a tab activates that artifact (loads its YAML into the editor). Close button on each tab removes the artifact from the workspace.

**Why tabs:** Users already understand tabs from IDEs. Tabs communicate "you have N things open" without requiring a separate sidebar panel.

### D3: Agent proposals add to workspace

**Choice:** `proposeArtifact()` still queues a proposal banner. `applyProposal()` now adds the artifact to the workspace (or updates if same name exists) and activates it. The workspace grows as agents produce output.

**Why not auto-add:** The user explicitly asked for approval gating. Proposals are suggestions; the workspace is the user's accepted state.

### D4: Context injection via message parts

**Choice:** When starting a new job, the user optionally selects artifacts from the workspace. Selected artifacts are serialized as additional `TextPart` entries in the A2A `message.parts` array, prefixed with a header like `--- Context: threat-catalog.yaml ---`.

**Why message parts:** This is native A2A protocol. No custom fields, no agent-side changes. The agent sees the context as part of the user's message, which is exactly how human experts would share artifacts — "Here's my threat catalog, now write a policy."

**Why not MCP tools:** Agents already have MCP tools for registry and GitHub. But workspace artifacts don't live in either — they're client state. Serializing into the prompt is the simplest path. If artifacts get large, we can add a `/api/workspace/{id}` endpoint later and give agents an MCP tool to fetch by reference.

### D5: Bundle publish extends existing dialog

**Choice:** The publish dialog already accepts `artifacts: Artifact[]`. The workspace passes all selected artifacts (defaulting to all). No backend changes — the `/api/publish` endpoint already bundles an array.

**Why no new endpoint:** The current publish handler iterates `artifacts[]` and bundles them. Multiple artifacts are already supported at the API level.

### D6: localStorage persistence

**Choice:** Workspace state serialized to `localStorage` under `complytime-studio-workspace`. Same pattern as `jobs.ts`. Re-hydrated on page load.

**Why not IndexedDB:** YAML artifacts are small (< 1 MB each). localStorage is synchronous and simpler. IndexedDB adds async complexity with no benefit at this scale.

## Risks / Trade-offs

| Risk | Mitigation |
|:--|:--|
| localStorage 5 MB limit with many artifacts | Display warning at 80% capacity. Future: server-side persistence. |
| Large artifacts in message parts bloat agent context window | Cap injected context at 100 KB total. Show warning if selection exceeds limit. |
| Tab bar becomes unwieldy with > 10 artifacts | Add horizontal scrolling with overflow indicator. Future: tree view sidebar. |
| Migration breaks existing editor imports | Thin wrappers in `editor.ts` re-export from `workspace.ts`. No component changes in phase 1. |
| Agent context injection changes prompt semantics | Prefix each injected artifact with clear delimiters. Agent prompts already handle multi-document YAML. |
