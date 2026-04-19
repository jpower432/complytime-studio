## 1. Fix Mission Dialog Bugs

- [x] 1.1 Replace `signal("")` and `signal(false)` with `useState` hooks in `NewMissionDialog` (`missions-view.tsx`)
- [x] 1.2 Remove duplicate `+ New Mission` button from empty-state block (keep header button only)
- [x] 1.3 Verify error message displays when A2A endpoint is unreachable

## 2. Fix Registry Browser Backend

- [x] 2.1 In `registryToolHandler` (`cmd/gateway/main.go`), validate MCP response text is valid JSON before writing; wrap non-JSON in `{"error": "<text>"}` and return HTTP 502
- [x] 2.2 Change `/api/registry/layer` handler tool name from `"fetch_manifest"` to `"fetch_layer"`

## 3. Agent Picker

- [x] 3.1 Add `agentName` field to `Mission` interface in `store/missions.ts`; default to `"studio-threat-modeler"` for existing records
- [x] 3.2 Create agent picker component in `NewMissionDialog` — fetch `/api/agents` on mount, render selectable agent cards
- [x] 3.3 Pass selected `agentName` through `sendMessage()` call in `handleSubmit`
- [x] 3.4 Store `agentName` in mission record via `createMission`
- [x] 3.5 Thread `agentName` from mission record into `streamTask()` and `sendReply()` in `MissionDetail`

## 4. Workspace Save Endpoint

- [x] 4.1 Add `POST /api/workspace/save` handler in `cmd/gateway/main.go` — accept `{filename, content}`, validate filename (reject `..`, absolute paths), write to `.complytime/artifacts/`
- [x] 4.2 Auto-create `.complytime/artifacts/` directory on first write
- [x] 4.3 Add `saveToWorkspace` function in a new `workbench/src/api/workspace.ts` client module

## 5. Save Actions in UI

- [x] 5.1 Add "Save" button to artifact panel toolbar (`artifact-panel.tsx`) — calls `saveToWorkspace` with artifact name and YAML content
- [x] 5.2 Add "Save to Workspace" button to registry browser layer view (`registry-browser.tsx`) — derives filename from repo name and media type
- [x] 5.3 Show brief success/error feedback after save operations
