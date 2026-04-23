## Context

The chat UI renders agent responses in two paths:

1. **Text messages** (`onMessage` → `streamBuffer` → finalized to `messages[]`) — rendered as markdown HTML via `renderMarkdown()`
2. **Artifact events** (`onArtifact` → `messages[]` with `.artifact` property) — rendered as artifact cards with "Save to Audit History" button

AuditLog YAML always arrives via path 1 because the ADK `save_artifact` in `after_agent` never fires. The artifact card UI (path 2) is dead code in practice.

## Goals / Non-Goals

**Goals:**
- User can save AuditLog YAML from chat text to audit history with one click
- Detection is automatic — no user action needed to surface the button
- Works for all valid AuditLog YAML in fenced code blocks

**Non-Goals:**
- Fixing the ADK `save_artifact` callback (remove instead)
- Auto-saving without user action (user controls what gets persisted)
- Detecting non-AuditLog artifacts (only AuditLog for now)

## Decisions

### D1: Detect at finalize time, not during streaming

**Choice:** Scan for AuditLog YAML when the message is finalized (in the `finalize` function), not during streaming.

**Why:** During streaming, the YAML block may be incomplete. Parsing partial YAML produces false negatives. At finalize time, the full text is available.

### D2: Extract YAML blocks and split into text + artifact messages

**Choice:** When an AuditLog is detected, split the agent message into: (1) the text portions as a regular message, and (2) each AuditLog as an artifact card message using the existing `ChatMessage.artifact` shape.

**Why:** Reuses the existing artifact card rendering and save button. No new UI components needed.

### D3: Parse with js-yaml in the browser

**Choice:** Use the existing YAML parsing approach (or add a lightweight parser) to check `metadata.type === "AuditLog"` on extracted code blocks.

**Alternative:** Regex check for `type: AuditLog`. Simpler but fragile — indentation, quoting, and comment variations would break it.

### D4: Remove broken `after_agent` artifact logic

**Choice:** Strip the AuditLog detection and `save_artifact` call from `callbacks.py`, leaving only the input validation and SQL guard.

**Why:** Dead code with zero successful executions. The detection is moving to the frontend where it actually works.

## Risks / Trade-offs

**[Risk] Large YAML blocks slow finalize** → Low risk. `yaml.safe_load` on a few KB YAML is sub-millisecond in JS. AuditLogs are typically 2-5KB.

**[Risk] False positive detection** → Mitigated by checking `metadata.type === "AuditLog"` after full YAML parse. Random YAML blocks without this field are ignored.

**[Risk] Agent returns YAML without fenced code block** → Accepted. The prompt instructs fenced blocks. Unfenced YAML is not detectable without ambiguity.
