## Why

The agent produces valid AuditLog YAML in chat responses, but users cannot persist them to audit history. The "Save to Audit History" button exists but only renders on `TaskArtifactUpdateEvent` SSE payloads. The ADK `save_artifact` callback never fires, so artifacts arrive as inline text — the button never appears.

The backend endpoint (`POST /api/audit-logs`) and the frontend save logic (`saveAuditLog()`) both exist and work. The gap is detection: the UI doesn't recognize AuditLog YAML in text messages.

## What Changes

- **Frontend detects AuditLog YAML in text messages** — after a message is finalized, scan for fenced YAML code blocks where the parsed content has `metadata.type: AuditLog`. Extract and render as artifact cards with the existing save button.
- **Remove dead `after_agent` artifact callback** — the ADK callback chain for `save_artifact` is broken and has zero log output. Remove the dead code path to avoid confusion. Keep the gateway interceptor for future use.

## Capabilities

### New Capabilities
- `inline-artifact-detection`: Detect and extract Gemara artifacts from agent text responses, rendering them as actionable cards

### Modified Capabilities
- `agent-spec-skills`: Remove broken `save_artifact` callback from `after_agent`

## Impact

- `workbench/src/components/chat-assistant.tsx` — add YAML extraction from finalized text, render artifact cards
- `workbench/src/lib/markdown.ts` — possibly extract YAML block parser for reuse
- `agents/assistant/callbacks.py` — simplify `after_agent` (remove artifact detection/save logic)
