## Why

The agent produces AuditLog YAML, but the current persistence path bypasses human review. The agent writes directly to `audit_logs` — the compliance record of truth — with no approval gate. This violates the principle that humans own audit assertions.

Real audit workflows are RFI-driven: an auditor receives a request, gathers evidence, reviews it, writes their report, and signs it. The agent should mirror this — assembling evidence and drafting classifications, while the human reviews, overrides, and promotes to official record.

**Security concern:** Giving an LLM write access to the compliance evidence store creates a DB grooming vector. The agent could fabricate classifications, selectively omit failing evidence, or normalize over time as humans stop reviewing. The agent must operate as an analyst, not an auditor.

## What Changes

- **Draft-first persistence** — Agent writes to `draft_audit_logs`, not `audit_logs`. Drafts carry per-result reasoning so the human can evaluate the agent's judgment.
- **Evidence Package artifact** — Before drafting, the agent emits a factual evidence package (no classifications, just data). This separates fact from judgment in the UI.
- **Human review gate** — The workbench renders draft results with Accept/Override/Note controls per result. The human promotes the draft to an official AuditLog.
- **Promote endpoint** — `POST /api/audit-logs/promote` moves a reviewed draft to `audit_logs` with the human's identity as the approver.
- **Revert direct POST** — Remove the agent's ability to write to `audit_logs`. The `publish_audit_log` tool writes drafts only.

## Capabilities

### New Capabilities
- `draft-audit-log`: Agent writes draft AuditLogs with per-result reasoning to a staging table
- `evidence-package`: Agent emits a factual evidence assembly as a read-only artifact
- `audit-review-gate`: Workbench renders draft results with Accept/Override/Note controls
- `promote-audit-log`: Gateway endpoint to promote a reviewed draft to official record

### Modified Capabilities
- `publish-audit-log-tool`: Targets `draft_audit_logs` instead of `audit_logs`. Includes reasoning per result.
- `agent-prompt`: Workflow splits into evidence assembly (factual) and draft classification (judgment)

### Removed Capabilities
- `agent-direct-persist`: Agent no longer POSTs to `audit_logs`. No service token needed.

## Impact

- `agents/assistant/tools.py` — `publish_audit_log` writes to `/api/draft-audit-logs`
- `agents/assistant/prompt.md` — two-phase workflow: evidence package → draft AuditLog
- `internal/store/handlers.go` — new `POST /api/draft-audit-logs`, `POST /api/audit-logs/promote`
- `internal/store/store.go` — `draft_audit_logs` table and promote logic
- `internal/clickhouse/migrations/` — DDL for `draft_audit_logs`
- `internal/auth/auth.go` — revert synthetic admin session for API tokens
- `workbench/` — draft review UI with per-result controls
