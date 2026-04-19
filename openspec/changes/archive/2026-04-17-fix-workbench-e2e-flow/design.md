## Context

The workbench is a Preact SPA embedded in the Go gateway binary. It communicates with backend services through the gateway's HTTP API:

```
Browser (Preact SPA)
  │
  ├── /api/a2a/{agent}    → reverse proxy to agent k8s service
  ├── /api/agents          → agent directory (from AGENT_DIRECTORY env)
  ├── /api/validate        → gemara-mcp proxy (MCP over streamable-http)
  ├── /api/publish         → OCI bundle assembly + oras push
  └── /api/registry/*      → oras-mcp proxy (MCP over streamable-http)
```

Five bugs and two missing features block the end-to-end flow. The bugs are in the Preact components and the gateway's registry proxy handler. The missing features are agent selection in mission creation and saving artifacts to the local filesystem.

## Goals / Non-Goals

**Goals:**
- Fix all bugs blocking the discover → create → validate → publish loop
- Add agent picker so missions aren't hardcoded to one specialist
- Add workspace save so users can pull artifacts locally from registry or missions

**Non-Goals:**
- Agent orchestration / multi-agent workflows
- Offline-first or local-only mode (gateway is required)
- Filesystem watch / auto-sync between `.complytime/` and the UI
- Registry authentication (handled separately by gateway proxy)

## Decisions

### D1: Replace signals with useState in NewMissionDialog

**Decision:** Convert `signal("")` / `signal(false)` local variables to `useState` hooks.

**Rationale:** Preact signals created as local `const` in a function component are re-instantiated on every render. When `submitting.value = true` triggers a re-render, new signal objects replace the old ones. The async catch block writes errors to the orphaned original signals — the UI never sees them.

`useState` hooks persist across re-renders by design. This is the standard Preact/React pattern for component-local state. The rest of the app already uses `useState` in `RegistryBrowser`, `ArtifactPanel`, and `PublishDialog`.

**Alternative considered:** `useSignal` from `@preact/signals` — would also work, but every other component in the workbench uses `useState` for local state. Consistency wins.

### D2: Remove duplicate "New Mission" button

**Decision:** Remove the `+ New Mission` button from the empty-state block (line 23), keep the one in the missions header (line 17).

**Rationale:** The header button is always visible. The empty-state button is redundant and creates visual clutter when the list is empty. Keeping the header button as the single entry point is consistent with the pattern used by the registry browser (single "Browse" button).

### D3: Agent picker in mission dialog

**Decision:** Fetch `/api/agents` when the dialog opens, display agent cards as selectable options, pass selected `agentName` through `sendMessage()` and `streamTask()`.

```
┌───────────────────────────────────────┐
│  New Mission                          │
│                                       │
│  Select Specialist:                   │
│  ┌─────────────────────────────────┐  │
│  │ ● studio-threat-modeler        │  │
│  │   Threat assessment & controls │  │
│  ├─────────────────────────────────┤  │
│  │ ○ studio-gap-analyst           │  │
│  │   Gap analysis                 │  │
│  ├─────────────────────────────────┤  │
│  │ ○ studio-policy-composer       │  │
│  │   Policy authoring             │  │
│  └─────────────────────────────────┘  │
│                                       │
│  Describe your mission:               │
│  ┌─────────────────────────────────┐  │
│  │                                 │  │
│  └─────────────────────────────────┘  │
│                                       │
│        [Cancel]  [Start Mission]      │
└───────────────────────────────────────┘
```

**Rationale:** The A2A layer already accepts `agentName` on every function (`sendMessage`, `sendReply`, `streamTask`). The `fetchAgents()` API client exists. The gateway serves `/api/agents`. Only the UI component is missing. The mission store needs a new `agentName` field so the detail view can route replies to the correct agent.

**Fallback:** If `/api/agents` returns empty or errors, default to `studio-threat-modeler` (current behavior) and show a note.

### D4: Fix registry proxy JSON error handling

**Decision:** In `registryToolHandler`, check if the MCP tool result text is valid JSON before writing it. If not, wrap it in `{"error": "<text>"}` and return 502.

**Rationale:** The MCP oras tools return plain text error messages (e.g., "invalid registry"). The handler sets `Content-Type: application/json` and writes the raw text. The frontend calls `res.json()` and gets `Unexpected token 'i'...`. The fix validates the response before forwarding.

### D5: Fix /api/registry/layer tool name

**Decision:** Change tool name from `"fetch_manifest"` to `"fetch_layer"` on the `/api/registry/layer` handler.

**Rationale:** Copy-paste bug. Line 139 of `cmd/gateway/main.go` registers the `/layer` endpoint but calls `fetch_manifest` instead of the correct oras-mcp tool.

### D6: Workspace save endpoint and UI

**Decision:** Add `POST /api/workspace/save` gateway endpoint. Writes artifact content to `.complytime/artifacts/<name>`. UI adds "Save" button in artifact panel and registry layer view.

```
POST /api/workspace/save
Content-Type: application/json

{
  "filename": "threat-catalog.yaml",
  "content": "metadata:\n  type: ThreatCatalog\n..."
}

→ writes to .complytime/artifacts/threat-catalog.yaml
→ returns { "path": ".complytime/artifacts/threat-catalog.yaml" }
```

**Rationale:** The browser can't write to disk. The gateway already runs on the user's machine (local dev) or in a pod (deployed). A simple write endpoint bridges the gap. The `.complytime/` directory already exists (in `.gitignore`). Scoping writes to `.complytime/artifacts/` prevents path traversal.

**Alternative considered:** Download via browser `<a download>` — works for single files but doesn't create a consistent workspace structure. Users want artifacts on disk for tooling integration, not just downloads.

**Security constraint:** The endpoint MUST validate that the resolved path stays within `.complytime/artifacts/`. Reject `..` traversal.

## Risks / Trade-offs

| Risk | Mitigation |
|:--|:--|
| `/api/workspace/save` is a filesystem write endpoint | Scope to `.complytime/artifacts/` only; validate paths; reject `..`; POST-only |
| Agent directory may be empty in local dev | Fallback to hardcoded default agent; show "No agents configured" message |
| MCP oras tool names may differ across versions | Verify tool name from `sess.ListTools()` at startup; log available tools |
| Mission store schema change (adding `agentName`) | Backward-compatible — existing localStorage missions without `agentName` default to `studio-threat-modeler` |
