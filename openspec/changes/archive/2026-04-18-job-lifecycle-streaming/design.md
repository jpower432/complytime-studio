## Context

The workbench manages agent interactions through a `Job` abstraction stored in localStorage. Today, a job is created when the user submits a prompt, an SSE stream connects to the gateway (which reverse-proxies to kagent), and status updates arrive as events. The chat panel renders completed messages and shows a static "Agent is working..." spinner during processing.

kagent's `A2aAgentExecutor` supports streaming via an `A2aAgentExecutorConfig.stream` field on the `DeclarativeAgentSpec`. When enabled, the ADK runner emits partial text events (token-by-token), `function_call` DataParts, and `function_response` DataParts — all of which flow through the existing SSE proxy. The Agent CRD also supports `requireApproval` on individual MCP tools, which pauses execution with `input-required` state until the user approves or rejects.

None of this infrastructure is wired into the workbench today. The chat panel ignores partial events, has no concept of tool calls, and the job lifecycle has no terminal user action.

## Goals / Non-Goals

**Goals:**
- Real-time visibility into agent execution (streaming text, tool calls, tool results)
- User control over tool execution via approve/reject for `requireApproval` tools
- Formal job acceptance with notes for audit context
- Clean job lifecycle: cancel active jobs, auto-purge history after 7 days
- Active/history split in the jobs view

**Non-Goals:**
- Server-side cancel (kagent's `cancel()` raises `NotImplementedError` — out of scope)
- Persistent server-side job storage (localStorage is sufficient for the workbench)
- Token-level streaming animation (word-level chunking is fine)
- Multi-user concurrent job support

## Decisions

### D1: Enable kagent streaming via Agent CRD `stream: true`

Add `stream: true` to each agent's `declarative` block in `agent-specialists.yaml`. This is a single-field change that switches ADK from buffered to streaming mode.

**Alternatives considered:**
- Gateway-injected heartbeat (only proves gateway is alive, not the agent)
- Gateway polling `tasks/get` (adds complexity, still no real-time text)

**Rationale:** Streaming solves both the "black hole" problem and enables tool call visibility with zero gateway changes. The gateway already proxies SSE with `FlushInterval: -1`.

### D2: Map kagent `completed` to workbench `ready`

The A2A protocol terminal state `completed` maps to a workbench-specific `ready` state, not removal. The user must explicitly accept the job.

| kagent State | Workbench State | User Action |
|:-------------|:----------------|:------------|
| `submitted` | `submitted` | Wait |
| `working` | `working` | Cancel available |
| `input-required` | `input-required` | Approve/reject tool, reply, or cancel |
| `completed` | `ready` | Accept (with note), iterate, or cancel |
| `failed` | `failed` | View error, delete |

**Rationale:** Compliance artifacts require human review. Auto-closing on agent completion skips the review step.

### D3: Client-side cancel only

Cancel closes the SSE EventSource, marks the job as `cancelled` in localStorage, and moves it to history. The agent continues on the server until it naturally completes.

**Alternatives considered:**
- A2A `tasks/cancel` (not implemented in kagent — `raise NotImplementedError`)
- Kill the agent pod (destructive, kills all sessions)

**Rationale:** Client-side cancel unblocks the user immediately. Token waste is acceptable for a dev/compliance workbench. Server-side cancel can be added when kagent implements it.

### D4: Streaming message accumulator pattern

Partial SSE events (`kagent.adk_partial: true`) append to a "live" message buffer in the chat. When `partial` flips to `false`, the buffer finalizes into a complete message. Tool call DataParts render as collapsible blocks interleaved with text.

Message types in the chat:

| Event Type | Rendering |
|:-----------|:----------|
| Partial text (`TextPart`, `adk_partial: true`) | Append to live bubble with typing cursor |
| Complete text (`TextPart`, `adk_partial: false`) | Finalize bubble |
| Function call (`DataPart`, type `function_call`) | Collapsible tool block showing name + args |
| Function call with `is_long_running: true` | Tool block with Approve / Reject buttons |
| Function response (`DataPart`, type `function_response`) | Update tool block with result, auto-collapse |

**Rationale:** Matches the mental model of watching someone work — text appears as they think, tool usage is visible but not noisy.

### D5: Accept with note, 7-day history, auto-purge

On acceptance, a dialog collects an optional note. The job moves to history state `accepted` with the note and timestamp. History entries auto-purge after 7 days via a cleanup check on app load and hourly interval.

**Storage model change:**
```typescript
interface Job {
  // existing fields...
  acceptedAt?: string;     // ISO timestamp
  acceptNote?: string;     // user-provided context
}
```

**Purge logic:** On `app.tsx` mount and every 60 minutes, filter history jobs where `acceptedAt` or `updatedAt` (for cancelled) is older than 7 days. Remove from localStorage.

**Rationale:** Option C from exploration — notes provide audit context ("Shipped to ghcr.io/...", "Partial — revisiting next sprint") without adding friction. 7-day window is long enough to reference recent work, short enough to prevent localStorage bloat.

### D6: Jobs view split into Active and Recent

The jobs list splits into two sections with distinct affordances:

- **Active**: Jobs in `submitted`, `working`, `input-required`, or `ready`. Clicking navigates to workspace. Cancel available on all. Accept available on `ready`.
- **Recent**: Jobs in `accepted` or `cancelled`. Read-only — click to view artifacts/conversation. Delete available.

Empty states: "No active jobs" with prominent New Job button; "No recent history" is hidden entirely.

## Risks / Trade-offs

**[Token waste on cancel]** → Acceptable for dev workbench. Document that server-side cancel is a future kagent dependency. Monitor via `usage_metadata` in SSE events.

**[localStorage size with streaming messages]** → Streaming generates many small messages. Mitigate by coalescing finalized partial chunks into single messages in `addMessage`. Only store the final text, not each chunk.

**[requireApproval UX confusion]** → Users may not understand why the agent paused. Mitigate with clear visual distinction on tool blocks that need approval — different background color, explicit "Waiting for your approval" label.

**[SSE reconnection during streaming]** → The existing 5-attempt reconnect logic in `streamTask` may miss partial context. Mitigate by re-fetching full task state on reconnect via `tasks/get` if available, or accepting the gap with a "Reconnected — some messages may be missing" notice.

## Open Questions

None — all decisions resolved during exploration.
