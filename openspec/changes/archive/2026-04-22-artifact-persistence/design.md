## Context

The A2A proxy in `internal/agents/agents.go` is a `httputil.ReverseProxy` that passes SSE bytes between the browser and the agent. It does not inspect response content. The `event_converter.py` in the agent already enriches artifact parts with `mimeType`, `model`, `promptVersion`, and `name` metadata. The gateway's `createAuditLogHandler` already validates content via `ParseAuditLog` and stores to ClickHouse.

The browser currently acts as middleware: it receives the SSE stream, extracts artifacts, and offers a manual save button. This is the only path to persistence.

## Goals / Non-Goals

**Goals:**
- AuditLog artifacts persist to ClickHouse without browser intervention
- Provenance metadata (`model`, `prompt_version`) flows through automatically
- Manual save remains available as an idempotent confirm/re-save
- Feature is toggle-able per deployment

**Non-Goals:**
- Persisting non-AuditLog artifacts (future scope)
- Modifying the A2A protocol or agent behavior
- Replacing the frontend save flow (it becomes secondary, not removed)

## Decisions

### 1. Interception point: ResponseWriter wrapper on the A2A proxy

Wrap the `httputil.ReverseProxy` response writer with an `artifactInterceptor` that tees SSE data. For each line matching `event: artifact`, parse the JSON payload, check for `mimeType: application/yaml`, and call the store.

**Alternative considered:** Separate sidecar that consumes the SSE stream independently. Rejected — adds a component, duplicates the stream, and requires its own auth/store connection. The gateway already has both.

**Alternative considered:** Agent POSTs directly to `/api/audit-logs`. Rejected — couples the agent to the gateway API, requires the agent to hold an API token, and breaks the A2A abstraction.

### 2. Policy ID resolution

The A2A `TaskArtifactUpdateEvent` does not carry `policy_id`. Three resolution strategies:

| Strategy | Mechanism | Trade-off |
|:---|:---|:---|
| Parse from YAML content | `ParseAuditLog` already extracts `target.id` | Works if the AuditLog references a policy; fails if omitted |
| Context from initial message | Intercept the request body on the initial `POST /api/a2a/` and extract `policy_id` from the user message or injected dashboard context | Requires request-side parsing; fragile if context format changes |
| Default + override | Use `DEFAULT_POLICY_ID` env var; frontend re-save can correct | Simple; may store with wrong association |

**Decision:** Parse from YAML content first (the `framework` field is already extracted by `ParseAuditLog`). If `policy_id` cannot be derived, use the user's session context (the proxy already has access to the auth session). Fall back to `"unassigned"` and log a warning.

### 3. Deduplication

`ReplacingMergeTree(created_at)` on `audit_logs` means rows with the same `ORDER BY` key are deduplicated at merge time. The `audit_id` is generated server-side (`uuid.New()`). Two saves of the same content produce two rows with different `audit_id` values — they are **not** deduplicated.

**Decision:** Hash the artifact content (`sha256(content)[:16]`) and use it as `audit_id` for auto-persisted artifacts. This means:
- Auto-persist of the same YAML content is idempotent
- Manual re-save with the same content hits the same `audit_id` → deduplicated by `ReplacingMergeTree`
- Manual save with edited content gets a new hash → new row

### 4. SSE parsing approach

A2A SSE events follow the standard format: `event: <type>\ndata: <json>\n\n`. The interceptor reads line-by-line, buffers `data:` lines when `event:` matches artifact types, and parses complete events.

**Decision:** Use a streaming line scanner on the teed response bytes. Only parse events where `event:` contains `artifact`. Pass all bytes through to the client unchanged — the interceptor is read-only on the wire.

### 5. Feature toggle

**Decision:** `AUTO_PERSIST_ARTIFACTS` env var (string `"true"` / `"false"`, default `"true"`). Checked once at startup, stored in `agents.Options`. When disabled, the proxy behaves exactly as it does today.

## Risks / Trade-offs

| Risk | Mitigation |
|:---|:---|
| SSE parsing adds latency to the stream | Line-by-line scan with goroutine; store call is async (fire-and-forget with error logging). Stream delivery is not blocked. |
| Malformed SSE from agent | Interceptor only acts on successfully parsed artifact events. Malformed data passes through to the client unchanged. |
| `ParseAuditLog` rejects agent output | Log warning, skip persistence. The frontend still receives the artifact and can display it. User can edit and manually save. |
| Content hash collision | SHA-256 truncated to 16 hex chars (64 bits). Collision probability is negligible for audit log volumes. |
| `policy_id` derivation fails | Falls back to `"unassigned"` with a logged warning. User can correct via manual re-save. |
