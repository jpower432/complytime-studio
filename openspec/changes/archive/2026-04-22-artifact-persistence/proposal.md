## Why

Agent-produced AuditLog artifacts only reach ClickHouse if the user clicks "Save to Audit History" in the browser. The save path is: agent → A2A SSE stream → frontend `onArtifact` → user click → `POST /api/audit-logs`. If the tab closes, the stream errors, or the user forgets, the artifact is lost. This contradicts the "agentic audit-log synthesis" goal — the system promises automated analysis but requires a manual step for durability.

The gateway already proxies the A2A stream (`/api/a2a/{agent}`). It can inspect `TaskArtifactUpdateEvent` SSE events in-flight and persist `application/yaml` artifacts server-side without modifying the agent or requiring the browser to act as middleware.

## What Changes

- Add an artifact-aware A2A response interceptor in the gateway that detects `TaskArtifactUpdateEvent` with `mimeType: application/yaml` and persists the content to the audit-log store
- Extract `model` and `promptVersion` from artifact part metadata (already set by `event_converter.py`) and forward to `InsertAuditLog`
- Derive `policy_id` from the A2A task context or fall back to a configurable default
- Keep the frontend "Save to Audit History" button as an explicit re-save / confirm action (idempotent via `ReplacingMergeTree`)
- Add a gateway config flag `AUTO_PERSIST_ARTIFACTS` (default `true`) to allow operators to disable server-side persistence

## Capabilities

### New Capabilities
- `artifact-server-persist`: Gateway-side interception and persistence of agent-produced AuditLog artifacts from the A2A stream

### Modified Capabilities
- `streaming-chat`: Frontend save button becomes a confirmation action, not the primary persistence path

## Impact

- **Backend**: `internal/agents/` — new response interceptor wrapping the reverse proxy; `internal/store/` — reuse existing `InsertAuditLog`
- **Frontend**: `chat-assistant.tsx` — save button label/behavior change (confirm vs primary save); artifact card shows "Auto-saved" indicator
- **Agent**: No changes — provenance metadata already flows via `event_converter.py`
- **Helm**: New env var `AUTO_PERSIST_ARTIFACTS` in gateway deployment

## Constitution Alignment

### I. Autonomous Collaboration

**Assessment**: PASS

Artifacts become self-persisting — the agent produces, the gateway stores. No human in the loop for durability. The artifact remains self-describing (Gemara YAML with provenance metadata).

### II. Composability First

**Assessment**: PASS

The interceptor reuses the existing `InsertAuditLog` store method and `ParseAuditLog` validation. No new storage engine or schema. The feature is toggle-able via `AUTO_PERSIST_ARTIFACTS`.

### III. Observable Quality

**Assessment**: PASS

Every auto-persisted artifact carries `model` and `prompt_version` provenance. The gateway logs each auto-persist event. `ReplacingMergeTree` deduplication means manual re-save is safe.

### IV. Testability

**Assessment**: PASS

The interceptor is a pure function: SSE event bytes in → store call out. Testable with synthetic SSE streams and a mock `AuditLogStore`.
