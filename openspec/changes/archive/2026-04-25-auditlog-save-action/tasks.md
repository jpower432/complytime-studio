## 1. Draft Table & Internal Endpoint

- [x] 1.1 Add `draft_audit_logs` migration (same schema as `audit_logs` + `status` Enum8('pending_review','promoted','expired'), `agent_reasoning` String, `reviewed_by` Nullable(String), `promoted_at` Nullable(DateTime64))
- [x] 1.2 Add `DraftAuditLogStore` interface and ClickHouse implementation
- [x] 1.3 Add `POST /internal/draft-audit-logs` handler (no auth, cluster-internal)
- [x] 1.4 Add `GET /api/draft-audit-logs` handler (authenticated, lists pending drafts)
- [x] 1.5 Add `POST /api/audit-logs/promote` handler (admin required, copies draft → audit_logs, sets created_by to session user)

## 2. Agent Tool Update

- [x] 2.1 Rewrite `publish_audit_log` to POST to `/internal/draft-audit-logs` instead of `/api/audit-logs`
- [x] 2.2 Remove `STUDIO_API_TOKEN` env var from assistant deployment
- [x] 2.3 Remove `GATEWAY_TOKEN` / auth header logic from `tools.py`

## 3. Prompt Update

- [x] 3.1 Split agent workflow into two phases: evidence assembly (factual) → draft classification (judgment)
- [x] 3.2 Add `agent-reasoning` field requirement to AuditLog template per result
- [x] 3.3 Instruct agent to emit evidence package artifact before drafting

## 4. Auth Revert

- [x] 4.1 Remove synthetic admin session for API token requests in `auth.go`
- [x] 4.2 Restore API token to auth-bypass-only (no admin escalation)

## 5. Network Isolation

- [x] 5.1 Add NetworkPolicy restricting `/internal/*` ingress to `studio-assistant` pods
- [x] 5.2 Bind internal endpoints to cluster-internal listener or path-based restriction

## 6. Workbench Review UI

- [x] 6.1 Add "Pending Drafts" view listing draft AuditLogs
- [x] 6.2 Render per-result cards with classification, reasoning, and evidence refs
- [x] 6.3 Add Accept / Override / Add Note controls per result
- [x] 6.4 Add "Promote to Official" action that calls `POST /api/audit-logs/promote`
- [x] 6.5 Show promoted AuditLogs in existing audit history with `created_by` = human

## 7. Cleanup

- [x] 7.1 Remove `AutoPersistArtifacts` SSE interceptor logic (no longer needed)
- [x] 7.2 Remove `artifact_interceptor.go` and test file
- [x] 7.3 Add TTL or cleanup job for drafts older than 30 days

## 8. Verification

- [x] 8.1 Agent produces evidence package artifact (factual, no classifications)
- [x] 8.2 Agent produces draft AuditLog with per-result reasoning
- [x] 8.3 Draft appears in workbench review queue
- [x] 8.4 Human overrides one classification and promotes
- [x] 8.5 Official AuditLog in `audit_logs` has human as `created_by`
- [x] 8.6 Agent cannot reach `POST /api/audit-logs` or `POST /api/audit-logs/promote`
- [x] 8.7 Internal endpoint unreachable from non-assistant pods
