## Why

ComplyTime Studio agents are islands. Each job runs one agent, produces one artifact, and ends. To build a threat catalog then derive a policy from it, the user manually copies YAML between jobs. This makes the platform a chat app that happens to produce artifacts, not an agentic workflow platform. The core workflow — direct agents, review output, chain results, ship a bundle — requires the platform to connect agent outputs to agent inputs and treat related artifacts as a cohesive package.

## What Changes

- **Artifact workspace**: Jobs accumulate artifacts into a shared workspace (in-memory artifact map, not filesystem). Any artifact from any job is accessible to any agent in a subsequent job. The workspace replaces the single-editor signal as the source of truth.
- **Agent chaining**: When starting a new job, the user can select artifacts from the workspace as input context. The selected artifacts are included in the agent's initial prompt. "Take this threat catalog and write a policy" becomes a one-click handoff, not copy-paste.
- **Bundle publish**: Publish all workspace artifacts (or a selected subset) as a single OCI bundle. Replaces the current single-artifact publish path.
- **Workspace persistence**: Workspace state survives page refresh via localStorage (same pattern as jobs). Future: persist to server or Git.

## Capabilities

### New Capabilities
- `artifact-workspace`: Shared in-memory artifact store that accumulates output across jobs. Replaces the single-editor signal for artifact management. Provides artifact list, selection, and tab-based review.
- `agent-context-injection`: Mechanism for attaching workspace artifacts as input context when starting a new job. Selected artifacts are serialized into the agent's initial A2A message.
- `bundle-publish`: Publish multiple workspace artifacts as a single OCI Gemara bundle. Extends the existing publish endpoint to accept the full workspace selection.

### Modified Capabilities
- `workspace-editor`: Editor becomes one view of the active artifact within the workspace, not the sole artifact holder. Artifact proposals from agents add to the workspace rather than targeting "the editor."
- `workspace-save`: Download/save operations apply to individual artifacts or the entire workspace bundle.

## Impact

- **Frontend**: `workbench/src/store/editor.ts` refactored into `workspace.ts`. New workspace panel component. Job creation dialog gains artifact selector. Publish dialog gains multi-artifact support.
- **Backend**: `/api/publish` already accepts `artifacts[]` — no gateway change needed for multi-artifact publish.
- **Agent protocol**: No A2A protocol changes. Context injection uses the existing `message.parts` array — workspace artifacts are appended as additional text parts in the initial message.
- **Data model**: `Job` gains `workspaceArtifacts: string[]` (IDs of artifacts fed as input). Workspace store is a new signal-based store.
