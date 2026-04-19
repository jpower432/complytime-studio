## Why

The workbench has two disconnected content experiences: the mission detail view (chat + CodeMirror editor) and the registry browser (read-only `<pre>` dump). Users cannot move fluidly between discovering existing artifacts in a registry and authoring new ones through agent-assisted missions. The editor — the core tool for working with Gemara YAML — is buried inside the mission detail view and inaccessible from anywhere else. The registry browser has no way to feed discovered artifacts into the editing flow as `mapping-references`.

## What Changes

- **Promote the editor to a first-class, default view** — the YAML editor with validate/save/publish toolbar becomes the main workspace, always accessible, not gated behind a mission
- **Add registry import action** — browsing a registry and selecting a layer injects a `mapping-references` entry into the active editor artifact, using the imported artifact's `metadata.id`, `metadata.version`, `title`, and the OCI reference as `url`
- **Connect missions to the workspace editor** — when a mission produces artifacts, they populate the same central editor; chat becomes a side panel to the editor rather than the editor being a side panel to chat
- **Remove the standalone registry browser view** — registry browsing becomes an action within the workspace (e.g., an import dialog or side panel) rather than a separate top-level view

## Capabilities

### New Capabilities
- `workspace-editor`: First-class YAML editor view as the default/main view — supports loading content from missions, registry imports, and local files; hosts the validate/save/publish toolbar
- `registry-import`: Import action that pulls a registry artifact's metadata into the editor's `mapping-references` block — parses the imported YAML for `metadata.id`, `metadata.version`, `title`, constructs OCI URL from registry coordinates, and injects the reference entry

### Modified Capabilities

(none)

## Impact

| Area | Detail |
|:--|:--|
| `workbench/src/app.tsx` | New default view routing — editor becomes primary, missions/registry become supporting panels |
| `workbench/src/components/` | New workspace-editor component; mission-detail refactored so chat is a panel feeding the central editor; registry-browser becomes an import dialog |
| `workbench/src/components/sidebar.tsx` | Navigation updated — Editor (primary), Missions, Import |
| `workbench/src/store/missions.ts` | Mission artifacts feed into shared editor state |
| `workbench/src/lib/artifact-detect.ts` | New function to extract mapping-reference metadata from imported YAML |
| No backend changes | Editor state is client-side; registry API and workspace save already exist |
