## Why

The workbench UI advertises five capabilities — mission chat, validation, publishing, registry browsing, and agent selection — but only validation and publishing work end-to-end. Starting a mission fails silently (signal lifecycle bug), the registry browser chokes on non-JSON MCP responses, and the agent directory API is never called from the UI. Users cannot complete the core loop: discover existing artifacts, create new ones through agent-assisted missions, and publish bundles.

## What Changes

- **Fix mission dialog** — replace Preact signals-as-local-vars with `useState` hooks so errors surface; remove duplicate "New Mission" button in empty state
- **Add agent picker** — wire `fetchAgents()` into the mission dialog so users choose a specialist instead of always hitting `studio-threat-modeler`
- **Fix registry browser backend** — JSON-wrap MCP tool error text; fix `/api/registry/layer` calling wrong tool (`fetch_manifest` → `fetch_layer`)
- **Add "Save to workspace" action** — new gateway endpoint `POST /api/workspace/save` writes artifacts from the registry browser (or mission panel) to `.complytime/` on disk; UI gets a "Save" button on the manifest/layer view and the artifact panel

## Capabilities

### New Capabilities
- `agent-picker`: Agent directory integration in mission creation dialog — fetch available specialists, display cards, bind selection to A2A calls
- `workspace-save`: Save artifacts to local `.complytime/` directory from registry browser or mission artifact panel via gateway endpoint

### Modified Capabilities

(none — no existing specs)

## Impact

| Area | Detail |
|:--|:--|
| `workbench/src/components/missions-view.tsx` | Signal→hook refactor, agent picker integration, remove duplicate button |
| `workbench/src/components/registry-browser.tsx` | Add "Save" action on manifest/layer views |
| `workbench/src/components/artifact-panel.tsx` | Add "Save to workspace" button |
| `workbench/src/api/agents.ts` | Already exists, now consumed by UI |
| `workbench/src/api/registry.ts` | No change (backend fix) |
| `cmd/gateway/main.go` | Fix `registryToolHandler` JSON wrapping, fix `/layer` tool name, add `/api/workspace/save` endpoint |
| `.complytime/` | Directory created on first save; gitignored |
