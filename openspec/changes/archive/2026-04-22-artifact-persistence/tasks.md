## 1. Gateway: SSE Artifact Interceptor

- [x] 1.1 Create `internal/agents/artifact_interceptor.go` with `artifactInterceptor` struct wrapping `http.ResponseWriter`
- [x] 1.2 Implement `Write([]byte)` that tees SSE data to a line scanner goroutine while forwarding all bytes to the underlying writer unchanged
- [x] 1.3 Implement SSE event parser: buffer lines after `event:` containing `artifact`, parse `data:` JSON on blank-line boundary
- [x] 1.4 Extract artifact part `text` (YAML content) and `metadata` (`mimeType`, `model`, `promptVersion`, `name`) from parsed `TaskArtifactUpdateEvent`
- [x] 1.5 Compute `audit_id` as `sha256(content)[:16]` for content-addressed deduplication
- [x] 1.6 Derive `policy_id` from parsed AuditLog content; fall back to `"unassigned"` with warning log
- [x] 1.7 Call `InsertAuditLog` asynchronously (goroutine); log errors at `ERROR` level without blocking the stream

## 2. Gateway: Wiring and Config

- [x] 2.1 Add `AutoPersistArtifacts bool` field to `agents.Options`
- [x] 2.2 Read `AUTO_PERSIST_ARTIFACTS` env var in `cmd/gateway/main.go` (default `"true"`)
- [x] 2.3 When enabled, wrap the `httputil.ReverseProxy` response writer with `artifactInterceptor` in `registerA2AProxy`
- [x] 2.4 Pass `AuditLogStore` interface to `agents.Options` for the interceptor to call
- [x] 2.5 Add `AUTO_PERSIST_ARTIFACTS` to Helm `values.yaml` and gateway deployment template

## 3. Frontend: Auto-saved Indicator

- [x] 3.1 Add `autoSaved?: boolean` field to `ChatMessage.artifact` interface in `chat-assistant.tsx`
- [x] 3.2 Expose `AUTO_PERSIST_ARTIFACTS` flag via `/api/config` response
- [x] 3.3 When auto-persist is enabled, set `autoSaved: true` on artifact messages received via `onArtifact`
- [x] 3.4 Render "Auto-saved" indicator on artifact cards when `artifact.autoSaved` is true
- [x] 3.5 Keep "Save to Audit History" button visible for admins (idempotent re-save)

## 4. Tests

- [x] 4.1 Unit test: `artifactInterceptor` parses synthetic SSE stream with `TaskArtifactUpdateEvent` and calls mock `AuditLogStore`
- [x] 4.2 Unit test: `artifactInterceptor` forwards all bytes to underlying writer unchanged (no data loss)
- [x] 4.3 Unit test: non-YAML artifacts are ignored (no store call)
- [x] 4.4 Unit test: malformed SSE data passes through without error
- [x] 4.5 Unit test: `InsertAuditLog` error is logged but does not block the writer
- [x] 4.6 Unit test: content-addressed `audit_id` is deterministic for same content
- [x] 4.7 Unit test: feature toggle disabled — no interception, proxy behaves as before
- [x] 4.8 `go build ./...` passes
- [x] 4.9 `go test -race ./...` passes
- [x] 4.10 TypeScript `tsc --noEmit` passes
- [x] 4.11 Helm lint and template pass
