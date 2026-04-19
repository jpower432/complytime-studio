## Why

Jobs have no lifecycle management. Once created, they cannot be cancelled or removed. The agent produces no visible progress during execution — users see "Agent is working..." for up to an hour with zero signal. When the agent finishes, the job sits in the list forever with no way to formally close it. There is no mechanism for users to approve or reject tool calls, making `requireApproval` gates unusable.

## What Changes

- **Enable kagent streaming** on all Agent CRDs (`stream: true`), switching from buffered single-response to real-time SSE events containing partial text, tool calls, and tool results.
- **Redesign the chat panel** to render streaming text (typing indicator), collapsible tool call blocks, and inline approve/reject buttons for `requireApproval`-gated tools.
- **Introduce a job acceptance flow** — agent completion maps to `ready` (not done); the user reviews output, optionally iterates, then explicitly accepts with a note. Accepted jobs move to a 7-day history with auto-purge.
- **Add cancel and delete** — client-side cancel (closes SSE, marks cancelled) at any active state; delete available on history items.
- **Split the jobs view** into Active (submitted, working, ready) and Recent (accepted, cancelled) sections.

## Capabilities

### New Capabilities
- `job-lifecycle`: Job state machine (submitted → working → ready → accepted), cancel, delete, 7-day history with auto-purge, acceptance notes
- `streaming-chat`: Real-time streaming text rendering, tool call/response blocks, approve/reject HITL gates in the chat panel

### Modified Capabilities
- `agent-picker`: The new job dialog must respect the lifecycle — disable creation when an active job exists (already enforced, no spec change needed)
- `workspace-editor`: Download/publish actions gated on `ready` status to prevent export of incomplete artifacts

## Impact

- **Helm chart** (`agent-specialists.yaml`): Add `stream: true` to each agent's `declarative` block
- **Workbench** (`store/jobs.ts`): Extended `Message` type, new `acceptJob`/`deleteJob`/`purgeHistory` functions, 7-day TTL cleanup
- **Workbench** (`components/chat-panel.tsx`): Streaming text accumulator, tool call blocks, HITL approve/reject, lifecycle controls
- **Workbench** (`components/chat-drawer.tsx`): SSE handler updated to process partial events and DataPart function calls
- **Workbench** (`components/jobs-view.tsx`): Active/history split, accept dialog, delete action
- **Gateway**: No changes — already proxies SSE with `FlushInterval: -1`
